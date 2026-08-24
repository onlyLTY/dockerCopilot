package module

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ref "github.com/distribution/reference"
	"github.com/docker/docker/api/types/registry"
)

const maxDockerAuthConfigSize int64 = 1 << 20

type registryCredentials struct {
	Basic   string
	Encoded string
}

type dockerAuthFile struct {
	Auths map[string]registry.AuthConfig `json:"auths"`
}

func credentialsForReference(imageReference string) (registryCredentials, error) {
	named, err := ref.ParseNormalizedNamed(imageReference)
	if err != nil {
		return registryCredentials{}, err
	}
	domain := ref.Domain(named)
	configData, err := loadDockerAuthConfig()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return registryCredentials{}, nil
		}
		return registryCredentials{}, err
	}
	if len(configData) == 0 {
		return registryCredentials{}, nil
	}
	var configFile dockerAuthFile
	if err := json.Unmarshal(configData, &configFile); err != nil {
		return registryCredentials{}, fmt.Errorf("解析 Docker registry 凭据失败: %w", err)
	}
	for server, authConfig := range configFile.Auths {
		if !registryServerMatches(server, domain) {
			continue
		}
		username, password, err := authUsernamePassword(authConfig)
		if err != nil {
			return registryCredentials{}, err
		}
		authConfig.Username = username
		authConfig.Password = password
		authConfig.ServerAddress = domain
		encoded, err := registry.EncodeAuthConfig(authConfig)
		if err != nil {
			return registryCredentials{}, err
		}
		basic := authConfig.Auth
		if username != "" || password != "" {
			basic = base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		}
		return registryCredentials{Basic: basic, Encoded: encoded}, nil
	}
	return registryCredentials{}, nil
}

func loadDockerAuthConfig() ([]byte, error) {
	if raw := strings.TrimSpace(os.Getenv("DOCKER_AUTH_CONFIG")); raw != "" {
		if int64(len(raw)) > maxDockerAuthConfigSize {
			return nil, errors.New("DOCKER_AUTH_CONFIG 超过大小限制")
		}
		return []byte(raw), nil
	}
	configDir := strings.TrimSpace(os.Getenv("DOCKER_CONFIG"))
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configDir = filepath.Join(home, ".docker")
	}
	configDir, err := filepath.Abs(configDir)
	if err != nil {
		return nil, err
	}
	configRoot, err := os.OpenRoot(configDir)
	if err != nil {
		return nil, err
	}
	defer configRoot.Close()
	file, err := configRoot.Open("config.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxDockerAuthConfigSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxDockerAuthConfigSize {
		return nil, errors.New("docker auth config 超过大小限制")
	}
	return content, nil
}

func registryServerMatches(server, domain string) bool {
	server = strings.TrimSpace(strings.ToLower(server))
	server = strings.TrimPrefix(server, "https://")
	server = strings.TrimPrefix(server, "http://")
	server = strings.TrimSuffix(server, "/v1/")
	server = strings.TrimSuffix(server, "/v2/")
	server = strings.TrimSuffix(server, "/")
	domain = strings.ToLower(domain)
	if server == domain {
		return true
	}
	return domain == DefaultRegistryDomain && (server == "index.docker.io" || server == "registry-1.docker.io")
}

func authUsernamePassword(authConfig registry.AuthConfig) (string, string, error) {
	if authConfig.Username != "" || authConfig.Password != "" {
		return authConfig.Username, authConfig.Password, nil
	}
	if authConfig.Auth == "" {
		return "", "", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(authConfig.Auth)
	if err != nil {
		return "", "", errors.New("docker registry auth 字段格式错误")
	}
	username, password, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return "", "", errors.New("docker registry auth 字段缺少密码分隔符")
	}
	return username, password, nil
}
