package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	socketPath    = "/run/dockercopilot-helper.sock"
	maxBody       = 32 * 1024
	operationTTL  = 10 * time.Minute
	maxOperations = 64
)

var (
	daemonPath = "/etc/docker/daemon.json"
	backupPath = "/var/lib/dockercopilot/daemon.json.bak"
)

type proxyValues struct {
	HTTPProxy  string `json:"httpProxy"`
	HTTPSProxy string `json:"httpsProxy"`
	NoProxy    string `json:"noProxy"`
}

type statusResponse struct {
	Enabled         bool        `json:"enabled"`
	FileExists      bool        `json:"fileExists"`
	Writable        bool        `json:"writable"`
	Hash            string      `json:"hash,omitempty"`
	FileProxy       proxyValues `json:"fileProxy"`
	DaemonProxy     proxyValues `json:"daemonProxy,omitempty"`
	RestartRequired bool        `json:"restartRequired"`
	Message         string      `json:"message,omitempty"`
}

type applyRequest struct {
	proxyValues
	Hash string `json:"hash"`
}

type applyResponse struct {
	statusResponse
	BackupCreated bool `json:"backupCreated"`
}

type restartResponse struct {
	OperationID string `json:"operationID"`
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
}

type helper struct {
	mu              sync.Mutex
	operations      map[string]*restartOperation
	restartingID    string
	restartRequired bool
}

type restartOperation struct {
	Status    string
	Message   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func main() {
	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		panic(err)
	}
	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	_ = os.Chmod(socketPath, 0660)

	h := &helper{operations: make(map[string]*restartOperation)}
	server := &http.Server{Handler: h.routes(), ReadHeaderTimeout: 5 * time.Second}
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

func (h *helper) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", h.status)
	mux.HandleFunc("/proxy/apply", h.apply)
	mux.HandleFunc("/daemon/restart", h.restart)
	mux.HandleFunc("/operations/", h.operation)
	return mux
}

func (h *helper) status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	status, err := h.readStatus()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *helper) apply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}
	var req applyRequest
	if err := decodeBody(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	if err := validateProxy(req.proxyValues); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	current, err := h.readStatus()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	if req.Hash != current.Hash && !(req.Hash == "" && !current.FileExists) {
		writeJSON(w, http.StatusConflict, map[string]string{"message": "daemon 配置已变化，请刷新后重试"})
		return
	}
	var content []byte
	if current.FileExists {
		content, err = os.ReadFile(daemonPath)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "无法读取 Docker daemon 配置"})
			return
		}
	} else {
		content = []byte(`{}`)
	}
	var doc map[string]any
	if err := json.Unmarshal(content, &doc); err != nil || doc == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "daemon.json 不是有效 JSON 对象"})
		return
	}
	proxies, _ := doc["proxies"].(map[string]any)
	if proxies == nil {
		proxies = map[string]any{}
	}
	setOptional(proxies, "http-proxy", req.HTTPProxy)
	setOptional(proxies, "https-proxy", req.HTTPSProxy)
	setOptional(proxies, "no-proxy", req.NoProxy)
	if len(proxies) == 0 {
		delete(doc, "proxies")
	} else {
		doc["proxies"] = proxies
	}
	updated, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "无法生成 daemon.json"})
		return
	}
	if err := writeConfig(content, updated); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	result, _ := h.readStatus()
	result.RestartRequired = true
	result.Message = "配置已写入，重启 Docker daemon 后生效"
	h.restartRequired = true
	writeJSON(w, http.StatusOK, applyResponse{statusResponse: result, BackupCreated: true})

}

func (h *helper) restart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.restartingID != "" {
		if operation, ok := h.operations[h.restartingID]; ok && operation.Status == "restarting" {
			writeJSON(w, http.StatusConflict, restartResponse{OperationID: h.restartingID, Status: operation.Status, Message: "Docker daemon 正在重启"})
			return
		}
		h.restartingID = ""
	}
	if err := validateDaemonConfig(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "daemon 配置校验失败，请先修正配置"})
		return
	}
	h.pruneOperationsLocked(time.Now())
	operationID := fmt.Sprintf("restart-%d", time.Now().UnixNano())
	now := time.Now()
	h.operations[operationID] = &restartOperation{Status: "restarting", Message: "正在重启 Docker daemon", CreatedAt: now, UpdatedAt: now}
	h.restartingID = operationID
	writeJSON(w, http.StatusAccepted, restartResponse{OperationID: operationID, Status: "restarting", Message: "正在重启 Docker daemon"})
	go h.runRestart(operationID)
}

func (h *helper) operation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/operations/")
	h.mu.Lock()
	h.pruneOperationsLocked(time.Now())
	operation, ok := h.operations[id]
	if ok {
		operation = &restartOperation{Status: operation.Status, Message: operation.Message, CreatedAt: operation.CreatedAt, UpdatedAt: operation.UpdatedAt}
	}
	h.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "operation not found"})
		return
	}
	writeJSON(w, http.StatusOK, restartResponse{OperationID: id, Status: operation.Status, Message: operation.Message})
}

func (h *helper) runRestart(operationID string) {
	cmdCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "systemctl", "restart", "docker")
	if err := cmd.Run(); err != nil {
		h.setOperation(operationID, "failed", "Docker daemon 重启失败")
		return
	}
	for i := 0; i < 15; i++ {
		if err := exec.CommandContext(cmdCtx, "docker", "info").Run(); err == nil {
			h.mu.Lock()
			h.restartRequired = false
			h.mu.Unlock()
			h.setOperation(operationID, "succeeded", "Docker daemon 已恢复")
			return
		}
		time.Sleep(time.Second)
	}
	h.setOperation(operationID, "failed", "Docker daemon 重启后未恢复")
}

func (h *helper) setOperation(id, status, message string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if operation, ok := h.operations[id]; ok {
		operation.Status = status
		operation.Message = message
		operation.UpdatedAt = time.Now()
		if h.restartingID == id {
			h.restartingID = ""
		}
	}
}

func (h *helper) pruneOperationsLocked(now time.Time) {
	for id, operation := range h.operations {
		if operation.Status != "restarting" && now.Sub(operation.UpdatedAt) > operationTTL {
			delete(h.operations, id)
		}
	}
	for len(h.operations) > maxOperations {
		var oldestID string
		var oldest time.Time
		for id, operation := range h.operations {
			if id == h.restartingID {
				continue
			}
			if oldestID == "" || operation.UpdatedAt.Before(oldest) {
				oldestID, oldest = id, operation.UpdatedAt
			}
		}
		if oldestID == "" {
			break
		}
		delete(h.operations, oldestID)
	}
}

func (h *helper) readStatus() (statusResponse, error) {
	status := statusResponse{Enabled: true, RestartRequired: h.restartRequired}
	content, err := os.ReadFile(daemonPath)
	if os.IsNotExist(err) {
		status.FileExists = false
		status.Writable = false
		return status, nil
	}
	if err != nil {
		return status, errors.New("无法读取 Docker daemon 配置")
	}
	info, err := os.Stat(daemonPath)
	if err != nil || !info.Mode().IsRegular() {
		return status, errors.New("Docker daemon 配置不是普通文件")
	}
	status.FileExists = true
	status.Writable = info.Mode().Perm()&0200 != 0
	status.Hash = hash(content)
	var doc map[string]any
	if err := json.Unmarshal(content, &doc); err != nil || doc == nil {
		return status, errors.New("daemon.json 不是有效 JSON 对象")
	}
	if proxies, ok := doc["proxies"].(map[string]any); ok {
		status.FileProxy = proxyValues{
			HTTPProxy:  stringValue(proxies["http-proxy"]),
			HTTPSProxy: stringValue(proxies["https-proxy"]),
			NoProxy:    stringValue(proxies["no-proxy"]),
		}
	}
	return status, nil
}

func writeConfig(old, updated []byte) error {
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return errors.New("无法创建 daemon 配置备份目录")
	}
	if err := os.WriteFile(backupPath, old, 0600); err != nil {
		return errors.New("无法备份 daemon 配置")
	}
	tmp, err := os.CreateTemp(filepath.Dir(daemonPath), ".daemon.json-*")
	if err != nil {
		return errors.New("无法创建 daemon 配置临时文件")
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return errors.New("无法设置 daemon 配置权限")
	}
	if _, err := tmp.Write(updated); err != nil {
		_ = tmp.Close()
		return errors.New("无法写入 daemon 配置")
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return errors.New("无法同步 daemon 配置")
	}
	if err := tmp.Close(); err != nil {
		return errors.New("无法关闭 daemon 配置")
	}
	if err := os.Rename(tmpName, daemonPath); err != nil {
		return errors.New("无法安全替换 daemon.json，请使用宿主机 helper 安装方式")
	}
	return nil
}

func validateDaemonConfig() error {
	content, err := os.ReadFile(daemonPath)
	if err != nil {
		return err
	}
	var doc map[string]any
	return json.Unmarshal(content, &doc)
}

func validateProxy(p proxyValues) error {
	for name, value := range map[string]string{"HTTP_PROXY": p.HTTPProxy, "HTTPS_PROXY": p.HTTPSProxy} {
		if value == "" {
			continue
		}
		u, err := url.Parse(value)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "socks5" && u.Scheme != "socks5h") || u.User != nil {
			return fmt.Errorf("%s 不是支持的代理 URL", name)
		}
	}
	if strings.ContainsAny(p.NoProxy, "\r\n") {
		return errors.New("NO_PROXY 不能包含换行")
	}
	return nil
}

func setOptional(values map[string]any, key, value string) {
	if value == "" {
		delete(values, key)
		return
	}
	values[key] = value
}

func decodeBody(r *http.Request, out any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	defer r.Body.Close()
	if err != nil || len(body) > maxBody {
		return errors.New("请求体过大或读取失败")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return errors.New("请求格式无效")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("请求只能包含一个 JSON 对象")
	}
	return nil
}

func hash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
