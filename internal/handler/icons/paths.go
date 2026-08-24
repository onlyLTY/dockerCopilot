package icons

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

const maxImageLogosConfigSize int64 = 1 << 20

var errImageLogosConfigTooLarge = errors.New("image logo config exceeds size limit")

var (
	imageUploadDir       = "/data/config/image"
	imageLogosPath       = "/data/config/imageLogos.json"
	legacyImageLogosPath = "/data/config/imageLogos.js"
)

func readImageLogosConfig(filePath string) ([]byte, error) {
	absolutePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(filepath.Dir(absolutePath))
	if err != nil {
		return nil, err
	}
	defer root.Close()
	name := filepath.Base(absolutePath)
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > maxImageLogosConfigSize {
		return nil, errImageLogosConfigTooLarge
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxImageLogosConfigSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxImageLogosConfigSize {
		return nil, errImageLogosConfigTooLarge
	}
	return content, nil
}
