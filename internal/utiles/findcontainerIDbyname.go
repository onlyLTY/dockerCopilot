package utiles

import (
	"errors"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
	"strings"
)

func findContainerIDByName(containers []types.Container, name string) (string, error) {
	for _, container := range containers {
		for _, containerName := range container.Names {
			trimmedName := strings.TrimPrefix(containerName, "/")
			if trimmedName == name {
				return container.ID, nil
			}
		}
	}
	return "", errors.New("container not found")
}
