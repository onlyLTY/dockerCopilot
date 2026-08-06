package icons

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	logicons "github.com/onlyLTY/dockerCopilot/internal/logic/icons"
)

func makeUploadRequest(t *testing.T, filename, contentType string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("imageName", "nginx:latest"); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/icons", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-Upload-Content-Type", contentType)
	return request
}

func withIconTestPaths(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	restore := logicons.SetIconPathsForTest(
		func() string { return filepath.Join(dir, "icons") },
		func() string { return filepath.Join(dir, "imageLogos.js") },
	)
	t.Cleanup(restore)
	return dir
}

func TestUploadRejectsContentThatDoesNotMatchExtension(t *testing.T) {
	dir := withIconTestPaths(t)
	recorder := httptest.NewRecorder()
	UploadHandler(nil)(recorder, makeUploadRequest(t, "logo.png", "image/png", []byte("not an image")))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if _, err := os.Stat(filepath.Join(dir, "icons")); !os.IsNotExist(err) {
		t.Fatalf("upload directory exists after rejected upload: %v", err)
	}
}

func TestUploadRejectsNonPNGExtension(t *testing.T) {
	withIconTestPaths(t)
	validPNG := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	recorder := httptest.NewRecorder()
	UploadHandler(nil)(recorder, makeUploadRequest(t, "logo.jpg", "image/jpeg", validPNG))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestUploadRejectsSVG(t *testing.T) {
	withIconTestPaths(t)
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	recorder := httptest.NewRecorder()
	UploadHandler(nil)(recorder, makeUploadRequest(t, "logo.svg", "image/svg+xml", svg))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestUploadStoresValidatedImageWithRandomFilename(t *testing.T) {
	dir := withIconTestPaths(t)
	validPNG := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 16)...)
	recorder := httptest.NewRecorder()
	UploadHandler(nil)(recorder, makeUploadRequest(t, "../../logo.png", "image/png", validPNG))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Code int    `json:"code"`
		Data string `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || response.Data == "" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if strings.Contains(response.Data, "logo") || !strings.HasSuffix(response.Data, ".png") {
		t.Fatalf("unexpected stored filename: %q", response.Data)
	}
	if _, err := os.Stat(filepath.Join(dir, "icons", response.Data)); err != nil {
		t.Fatalf("stored image missing: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "imageLogos.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), response.Data) {
		t.Fatalf("config does not reference %q: %s", response.Data, content)
	}
}
