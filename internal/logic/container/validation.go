package container

import (
	"errors"
	"regexp"
	"strings"
)

var containerNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

func validateContainerName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 128 || !containerNamePattern.MatchString(name) {
		return "", errors.New("容器名称格式错误")
	}
	return name, nil
}
