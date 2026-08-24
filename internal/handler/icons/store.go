package icons

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/onlyLTY/dockerCopilot/internal/imageref"
)

var legacyImageLogoEntryPattern = regexp.MustCompile(`"([^"]+)"\s*:\s*"([^"]+)"`)

func obtainImageLogos() (map[string]string, error) {
	imageLogosMu.Lock()
	defer imageLogosMu.Unlock()
	return loadImageLogosLocked()
}

func updateImageLogoMapping(imageName, filename string) (string, error) {
	key, err := imageref.RepositoryKey(imageName)
	if err != nil {
		return "", fmt.Errorf("invalid imageName: %w", err)
	}
	imageLogosMu.Lock()
	defer imageLogosMu.Unlock()
	logos, err := loadImageLogosLocked()
	if err != nil {
		return "", err
	}
	oldValue := logos[key]
	logos[key] = "/src/config/image/" + filename
	if err := writeImageLogosLocked(logos); err != nil {
		return "", err
	}
	return storedIconFilename(oldValue), nil
}

func loadImageLogosLocked() (map[string]string, error) {
	content, err := readImageLogosConfig(imageLogosPath)
	if err == nil {
		var logos map[string]string
		if err := json.Unmarshal(content, &logos); err != nil {
			return nil, fmt.Errorf("parse image logo config: %w", err)
		}
		if logos == nil {
			logos = make(map[string]string)
		}
		return logos, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	logos, migrationErr := readLegacyImageLogos()
	if migrationErr != nil {
		return nil, migrationErr
	}
	if err := writeImageLogosLocked(logos); err != nil {
		return nil, err
	}
	return logos, nil
}

func readLegacyImageLogos() (map[string]string, error) {
	logos := make(map[string]string)
	content, err := readImageLogosConfig(legacyImageLogosPath)
	if errors.Is(err, os.ErrNotExist) {
		return logos, nil
	}
	if err != nil {
		return nil, err
	}
	for _, match := range legacyImageLogoEntryPattern.FindAllStringSubmatch(string(content), -1) {
		if len(match) != 3 {
			continue
		}
		key, keyErr := imageref.RepositoryKey(match[1])
		if keyErr != nil {
			continue
		}
		logos[key] = match[2]
	}
	return logos, nil
}

func writeImageLogosLocked(logos map[string]string) (retErr error) {
	content, err := json.MarshalIndent(logos, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	configDir, err := filepath.Abs(filepath.Dir(imageLogosPath))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}
	root, err := os.OpenRoot(configDir)
	if err != nil {
		return err
	}
	defer root.Close()
	temporary, err := os.CreateTemp(configDir, ".image-logos-*")
	if err != nil {
		return err
	}
	temporaryName := filepath.Base(temporary.Name())
	defer func() {
		_ = temporary.Close()
		_ = root.Remove(temporaryName)
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return root.Rename(temporaryName, filepath.Base(imageLogosPath))
}

func storedIconFilename(value string) string {
	const prefix = "/src/config/image/"
	if !strings.HasPrefix(value, prefix) {
		return ""
	}
	filename := strings.TrimPrefix(value, prefix)
	if filename == filepath.Base(filename) && filename != "." {
		return filename
	}
	return ""
}
