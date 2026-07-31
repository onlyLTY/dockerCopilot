package icons

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var iconConfigMu sync.Mutex
var iconDirectory = func() string { return "/data/icon/icons" }
var iconConfigPath = func() string { return "/data/icon/imageLogos.js" }
var invalidRepositoryChars = regexp.MustCompile(`[^a-z0-9._:/-]+`)
var trailingComma = regexp.MustCompile(`,\s*}`)

func withIconConfigLock[T any](fn func() (T, error)) (T, error) {
	iconConfigMu.Lock()
	defer iconConfigMu.Unlock()
	return fn()
}

func normalizeRepository(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", fmt.Errorf("imageName is required")
	}
	if strings.ContainsAny(value, "\r\n\t \"'") {
		return "", fmt.Errorf("invalid imageName")
	}
	if at := strings.IndexByte(value, '@'); at >= 0 {
		value = value[:at]
	}
	lastSlash := strings.LastIndexByte(value, '/')
	if colon := strings.LastIndexByte(value, ':'); colon > lastSlash {
		value = value[:colon]
	}
	if strings.Contains(value, "..") || invalidRepositoryChars.MatchString(value) {
		return "", fmt.Errorf("invalid imageName")
	}
	parts := strings.Split(value, "/")
	if len(parts) > 1 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") || parts[0] == "localhost") {
		value = strings.Join(parts[1:], "/")
	}
	value = strings.Trim(value, "/")
	if value == "" {
		return "", fmt.Errorf("invalid imageName")
	}
	if invalidRepositoryChars.MatchString(value) || strings.Contains(value, "..") {
		return "", fmt.Errorf("invalid imageName")
	}
	return value, nil
}

func iconExtension(original string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filepath.Base(original)))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".svg", ".gif":
	default:
		return "", fmt.Errorf("仅支持 png、jpg、jpeg、webp、svg、gif 图标")
	}
	return ext, nil
}

func imageURL(filename string) string { return "/src/config/image/" + filename }

func readIcons(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	text := string(content)
	start := strings.IndexByte(text, '{')
	end := strings.LastIndexByte(text, '}')
	if start < 0 || end < start {
		return nil, fmt.Errorf("invalid config format")
	}
	jsonText := trailingComma.ReplaceAllString(text[start:end+1], "}")
	icons := make(map[string]string)
	if err := json.Unmarshal([]byte(jsonText), &icons); err != nil {
		return nil, fmt.Errorf("invalid config format: %w", err)
	}
	return icons, nil
}

func writeIcons(path string, icons map[string]string) error {
	keys := make([]string, 0, len(icons))
	for key := range icons {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	builder.WriteString("// 自定义镜像logo配置\nexport const customImageLogos = {\n")
	for _, key := range keys {
		keyJSON, _ := json.Marshal(key)
		valueJSON, _ := json.Marshal(icons[key])
		fmt.Fprintf(&builder, "  %s: %s,\n", keyJSON, valueJSON)
	}
	builder.WriteString("};\n")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".imageLogos-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(builder.String()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func matchingKeys(icons map[string]string, repository string) []string {
	keys := make([]string, 0)
	for key := range icons {
		if normalized, err := normalizeRepository(key); err == nil && normalized == repository {
			keys = append(keys, key)
		}
	}
	return keys
}
