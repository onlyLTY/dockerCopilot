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
	jsonPath := filepath.Join(tempDir, "imageLogos.json")
	imageDir := filepath.Join(tempDir, "image")
	testFilename := "codex-upload-vuln.json"

	originalImageUploadDir := imageUploadDir
	originalImageLogosPath := imageLogosPath
	originalLegacyImageLogosPath := legacyImageLogosPath
	imageUploadDir = imageDir
	imageLogosPath = jsonPath
	legacyImageLogosPath = filepath.Join(tempDir, "imageLogos.js")
	t.Cleanup(func() {
		imageUploadDir = originalImageUploadDir
		imageLogosPath = originalImageLogosPath
		legacyImageLogosPath = originalLegacyImageLogosPath
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

func TestValidateImageNameAcceptsDigestReferences(t *testing.T) {
	value := "registry.example/team/app@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := validateImageName(value); err != nil {
		t.Fatalf("valid digest reference was rejected: %v", err)
	}
	if err := validateImageName("not a valid image"); err == nil {
		t.Fatal("invalid image reference was accepted")
	}
}

func TestUploadHandlerRejectsOversizedImage(t *testing.T) {
	tempDir := t.TempDir()
	originalImageUploadDir := imageUploadDir
	originalImageLogosPath := imageLogosPath
	originalLegacyImageLogosPath := legacyImageLogosPath
	imageUploadDir = filepath.Join(tempDir, "image")
	imageLogosPath = filepath.Join(tempDir, "imageLogos.json")
	legacyImageLogosPath = filepath.Join(tempDir, "imageLogos.js")
	t.Cleanup(func() {
		imageUploadDir = originalImageUploadDir
		imageLogosPath = originalImageLogosPath
		legacyImageLogosPath = originalLegacyImageLogosPath
	})

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

func TestUpdateImageLogosJSONIsConcurrencySafe(t *testing.T) {
	tempDir := t.TempDir()
	originalImageLogosPath := imageLogosPath
	originalLegacyImageLogosPath := legacyImageLogosPath
	imageLogosPath = filepath.Join(tempDir, "imageLogos.json")
	legacyImageLogosPath = filepath.Join(tempDir, "imageLogos.js")
	t.Cleanup(func() {
		imageLogosPath = originalImageLogosPath
		legacyImageLogosPath = originalLegacyImageLogosPath
	})
	var waitGroup sync.WaitGroup
	errorsChannel := make(chan error, 10)
	for index := 0; index < 10; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			_, err := updateImageLogoMapping(fmt.Sprintf("image-%d:latest", index), fmt.Sprintf("%d.png", index))
			errorsChannel <- err
		}(index)
	}
	waitGroup.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Fatal(err)
		}
	}
	content, err := os.ReadFile(imageLogosPath)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 10; index++ {
		if !bytes.Contains(content, []byte(fmt.Sprintf(`"docker.io/library/image-%d"`, index))) {
			t.Fatalf("concurrent update lost image-%d: %s", index, content)
		}
	}
}

func TestLegacyImageLogosAreMigratedToJSON(t *testing.T) {
	tempDir := t.TempDir()
	originalImageLogosPath := imageLogosPath
	originalLegacyImageLogosPath := legacyImageLogosPath
	imageLogosPath = filepath.Join(tempDir, "imageLogos.json")
	legacyImageLogosPath = filepath.Join(tempDir, "imageLogos.js")
	t.Cleanup(func() {
		imageLogosPath = originalImageLogosPath
		legacyImageLogosPath = originalLegacyImageLogosPath
	})
	legacy := `export const customImageLogos = {"postgres:17-alpine": "/src/config/image/postgres.png"};`
	if err := os.WriteFile(legacyImageLogosPath, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	logos, err := obtainImageLogos()
	if err != nil {
		t.Fatal(err)
	}
	if logos["docker.io/library/postgres"] != "/src/config/image/postgres.png" {
		t.Fatalf("legacy mapping was not normalized: %+v", logos)
	}
	if _, err := os.Stat(imageLogosPath); err != nil {
		t.Fatalf("JSON migration was not persisted: %v", err)
	}
}
