package handler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 防回潮：routes 必须挂上 manual 补丁；logic 不得以 goctl todo stub 挂业务。
func TestRoutesCallRegisterManualHandlers(t *testing.T) {
	content, err := os.ReadFile("routes.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "RegisterManualHandlers") {
		t.Fatal("routes.go must call RegisterManualHandlers (SecurityHeaders + login rate limit)")
	}
}

func TestNoTodoLogicStubs(t *testing.T) {
	root := filepath.Join("..", "logic")
	var bad []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "todo: add your logic") {
			bad = append(bad, path)
		}
		return nil
	})
	if len(bad) > 0 {
		t.Fatalf("logic files still contain goctl todo stubs: %v", bad)
	}
}
