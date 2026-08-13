package daemonhelper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const DefaultSocketPath = "/run/dockercopilot-helper.sock"

type ProxyValues struct {
	HTTPProxy  string `json:"httpProxy"`
	HTTPSProxy string `json:"httpsProxy"`
	NoProxy    string `json:"noProxy"`
}

type Status struct {
	Enabled         bool        `json:"enabled"`
	FileExists      bool        `json:"fileExists"`
	Writable        bool        `json:"writable"`
	Hash            string      `json:"hash"`
	FileProxy       ProxyValues `json:"fileProxy"`
	RestartRequired bool        `json:"restartRequired"`
	Message         string      `json:"message,omitempty"`
}

type ApplyRequest struct {
	ProxyValues
	Hash string `json:"hash"`
}

type ApplyResponse struct {
	Status        Status `json:"status"`
	BackupCreated bool   `json:"backupCreated"`
}

type RestartResponse struct {
	OperationID string `json:"operationID"`
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
}

type Client struct {
	httpClient *http.Client
}

func New() *Client {
	path := strings.TrimSpace(os.Getenv("DOCKERCOPILOT_HELPER_SOCKET"))
	if path == "" {
		path = DefaultSocketPath
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "unix", path)
		},
	}
	return &Client{httpClient: &http.Client{Transport: transport, Timeout: 10 * time.Second}}
}

func (c *Client) Status(ctx context.Context) (Status, error) {
	var result Status
	if err := c.do(ctx, http.MethodGet, "/status", nil, &result); err != nil {
		return Status{}, err
	}
	return result, nil
}

func (c *Client) Apply(ctx context.Context, request ApplyRequest) (ApplyResponse, error) {
	var result ApplyResponse
	if err := c.do(ctx, http.MethodPost, "/proxy/apply", request, &result); err != nil {
		return ApplyResponse{}, err
	}
	return result, nil
}

func (c *Client) Restart(ctx context.Context) (RestartResponse, error) {
	var result RestartResponse
	if err := c.do(ctx, http.MethodPost, "/daemon/restart", map[string]string{}, &result); err != nil {
		return RestartResponse{}, err
	}
	return result, nil
}

func (c *Client) Operation(ctx context.Context, operationID string) (RestartResponse, error) {
	var result RestartResponse
	if err := c.do(ctx, http.MethodGet, "/operations/"+operationID, nil, &result); err != nil {
		return RestartResponse{}, err
	}
	return result, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		content, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(content)
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://helper"+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("helper 不可用")
	}
	defer response.Body.Close()
	var envelope struct {
		Message string `json:"message"`
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, 128*1024+1))
	if err != nil || len(content) > 128*1024 {
		return fmt.Errorf("helper 响应无效")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_ = json.Unmarshal(content, &envelope)
		if envelope.Message == "" {
			envelope.Message = "helper 操作失败"
		}
		return &HTTPError{StatusCode: response.StatusCode, Message: envelope.Message}
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(content, result); err != nil {
		return errors.New("helper 响应无效")
	}
	return nil
}

func readLimited(reader interface{ Read([]byte) (int, error) }, limit int64) ([]byte, error) {
	var result bytes.Buffer
	if _, err := result.ReadFrom(ioLimitReader{reader: reader, remaining: limit}); err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

type ioLimitReader struct {
	reader    interface{ Read([]byte) (int, error) }
	remaining int64
}

func (r ioLimitReader) Read(buffer []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, errors.New("response too large")
	}
	if int64(len(buffer)) > r.remaining {
		buffer = buffer[:r.remaining]
	}
	n, err := r.reader.Read(buffer)
	r.remaining -= int64(n)
	return n, err
}

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string { return e.Message }
