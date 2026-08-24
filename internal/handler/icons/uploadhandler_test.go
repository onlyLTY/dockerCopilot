package icons

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func TestUploadHandlerRejectsNonImageFiles(t *testing.T) {
	tempDir := t.TempDir()
	jsPath := filepath.Join(tempDir, "imageLogos.js")
	imageDir := filepath.Join(tempDir, "image")
	testFilename := "codex-upload-vuln.json"

	originalImageUploadDir := imageUploadDir
	originalImageLogosPath := imageLogosPath
	imageUploadDir = imageDir
	imageLogosPath = jsPath
	t.Cleanup(func() {
		imageUploadDir = originalImageUploadDir
		imageLogosPath = originalImageLogosPath
	})

	originalContent, readErr := os.ReadFile(jsPath)
	hadOriginal := readErr == nil
	if err := os.WriteFile(jsPath, []byte("// test\nexport const customImageLogos = {\n};\n"), 0o644); err != nil {
		t.Fatalf("failed to seed imageLogos.js: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(filepath.Join(imageDir, testFilename))
		if hadOriginal {
			_ = os.WriteFile(jsPath, originalContent, 0o644)
		} else {
			_ = os.Remove(jsPath)
		}
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileWriter, err := writer.CreateFormFile("file", testFilename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := io.WriteString(fileWriter, `{"not":"an image"}`); err != nil {
		t.Fatalf("failed to write form file: %v", err)
	}
	if err := writer.WriteField("imageName", "nginx"); err != nil {
		t.Fatalf("failed to write imageName: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/icons", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	UploadHandler(&svc.ServiceContext{})(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-image upload, got %d with body %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(imageDir, testFilename)); !os.IsNotExist(err) {
		t.Fatalf("expected non-image upload to be rejected without writing file, stat err=%v", err)
	}
}

func TestUploadHandlerRejectsOversizedImage(t *testing.T) {
	tempDir := t.TempDir()
	originalImageUploadDir := imageUploadDir
	originalImageLogosPath := imageLogosPath
	imageUploadDir = filepath.Join(tempDir, "image")
	imageLogosPath = filepath.Join(tempDir, "imageLogos.js")
	t.Cleanup(func() {
		imageUploadDir = originalImageUploadDir
		imageLogosPath = originalImageLogosPath
	})
	if err := os.WriteFile(imageLogosPath, []byte("export const customImageLogos = {\n};\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileWriter, err := writer.CreateFormFile("file", "large.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fileWriter.Write(bytes.Repeat([]byte{'x'}, int(maxImageFileSize+1))); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("imageName", "large"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/icons", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	UploadHandler(&svc.ServiceContext{})(recorder, req)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestUpdateImageLogosJSIsConcurrencySafe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "imageLogos.js")
	if err := os.WriteFile(path, []byte("export const customImageLogos = {\n};\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var waitGroup sync.WaitGroup
	errorsChannel := make(chan error, 10)
	for index := 0; index < 10; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			errorsChannel <- updateImageLogosJS(path, fmt.Sprintf("image-%d", index), fmt.Sprintf("%d.png", index))
		}(index)
	}
	waitGroup.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Fatal(err)
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 10; index++ {
		if !bytes.Contains(content, []byte(fmt.Sprintf(`"image-%d"`, index))) {
			t.Fatalf("concurrent update lost image-%d: %s", index, content)
		}
	}
}
