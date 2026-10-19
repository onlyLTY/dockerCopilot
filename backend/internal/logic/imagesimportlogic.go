package logic

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	dockerMsgType "github.com/docker/docker/pkg/jsonmessage"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// ErrImportTooLarge 上传的镜像包超过配置上限。
// handler 把 MaxBytesReader 的错误映射为该哨兵，便于返回明确的业务文案。
var ErrImportTooLarge = errors.New("镜像包超过大小限制")

// ImportInput 已解析的上传流；Reader 直接从网络读取，不在内存中缓存镜像包。
type ImportInput struct {
	Filename string
	Reader   io.Reader
}

type importResult struct {
	Loaded []string `json:"loaded"`
	Count  int      `json:"count"`
}

type ImportImagesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewImportImagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ImportImagesLogic {
	return &ImportImagesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Import 把 docker save（可 gzip/bzip2 压缩）的镜像包流式导入本机 Docker。
// 上传、解压、落盘在同一个请求内完成，镜像包不会写入应用数据目录。
func (l *ImportImagesLogic) Import(in ImportInput) (*types.Resp, error) {
	resp := &types.Resp{}
	if err := l.svcCtx.RequireDocker(); err != nil {
		return Fail(resp, err, 500, "导入镜像失败")
	}
	if in.Reader == nil {
		return Biz(400, "请选择要导入的镜像包", nil), nil
	}
	if !l.svcCtx.AcquireImageOp(l.ctx) {
		if err := l.ctx.Err(); err != nil {
			return Fail(resp, err, 500, "导入已取消")
		}
		return Biz(409, "已有镜像操作正在进行，请稍后重试", nil), nil
	}
	defer l.svcCtx.ReleaseImageOp()

	reader, closeReader, err := decompressImageArchive(in.Reader)
	if err != nil {
		if isTooLargeError(err) {
			return Biz(413, fmt.Sprintf("镜像包超过大小限制（最大 %s）", humanSize(l.svcCtx.Config.ImageImportLimit())), nil), nil
		}
		return Biz(400, "镜像包解压失败："+err.Error(), nil), nil
	}
	defer closeReader()

	l.Infof("开始导入镜像包 %s", in.Filename)
	loadResp, err := l.svcCtx.DockerClient.ImageLoad(l.ctx, reader)
	if err != nil {
		return l.importFailure(err)
	}
	defer loadResp.Body.Close()

	loaded, err := readImageLoadStream(loadResp.Body)
	if err != nil {
		return l.importFailure(err)
	}
	l.Infof("镜像包 %s 导入完成，共 %d 个镜像", in.Filename, len(loaded))
	if len(loaded) == 0 {
		return Biz(400, "镜像包中未找到可导入的镜像", nil), nil
	}
	return Ok(resp, importResult{Loaded: loaded, Count: len(loaded)}), nil
}

func (l *ImportImagesLogic) importFailure(err error) (*types.Resp, error) {
	resp := &types.Resp{}
	if isTooLargeError(err) {
		return Biz(413, fmt.Sprintf("镜像包超过大小限制（最大 %s）", humanSize(l.svcCtx.Config.ImageImportLimit())), nil), nil
	}
	if l.ctx.Err() != nil {
		return Fail(resp, l.ctx.Err(), 500, "导入已取消")
	}
	message := summarizeImportError(err)
	l.Errorf("导入镜像失败: %v", err)
	return Fail(resp, err, 500, "导入镜像失败："+message)
}

// isTooLargeError 识别 http.MaxBytesReader 及 go-zero MaxBytesHandler 的超限错误。
func isTooLargeError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrImportTooLarge) {
		return true
	}
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "request body too large") ||
		strings.Contains(lower, "request entity too large")
}

// decompressImageArchive 按魔数识别 gzip/bzip2；docker CLI 会在上传前解压，
// 这里兼容用户直接上传 .tar.gz/.tar.bz2 的情况。
func decompressImageArchive(reader io.Reader) (io.Reader, func(), error) {
	buffered := bufio.NewReader(reader)
	header, err := buffered.Peek(4)
	if err != nil && err != io.EOF {
		if errors.Is(err, ErrImportTooLarge) {
			return nil, nil, ErrImportTooLarge
		}
		return nil, nil, fmt.Errorf("读取镜像包失败")
	}
	switch {
	case len(header) >= 2 && header[0] == 0x1f && header[1] == 0x8b:
		gz, err := gzip.NewReader(buffered)
		if err != nil {
			return nil, nil, fmt.Errorf("不是有效的 gzip 文件")
		}
		return gz, func() { _ = gz.Close() }, nil
	case len(header) >= 3 && string(header[:3]) == "BZh":
		return bzip2.NewReader(buffered), func() {}, nil
	default:
		return buffered, func() {}, nil
	}
}

// readImageLoadStream 读取 Docker daemon 的加载输出。
// 成功时 stream 形如 "Loaded image: nginx:latest"，失败放在 errorDetail。
// 返回值为镜像引用列表（去掉 "Loaded image:" 前缀，无标签镜像保留为 ID）。
func readImageLoadStream(body io.Reader) ([]string, error) {
	decoder := json.NewDecoder(body)
	loaded := make([]string, 0)
	seen := make(map[string]struct{})
	for {
		var msg dockerMsgType.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if msg.Error != nil {
			if msg.Error.Message != "" {
				return nil, fmt.Errorf("%s", msg.Error.Message)
			}
			return nil, fmt.Errorf("Docker 返回未知错误")
		}
		if msg.ErrorMessage != "" {
			return nil, fmt.Errorf("%s", msg.ErrorMessage)
		}
		for _, line := range strings.Split(msg.Stream, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if _, ok := seen[line]; ok {
				continue
			}
			seen[line] = struct{}{}
			loaded = append(loaded, normalizeLoadedImageLine(line))
		}
	}
	return loaded, nil
}

func normalizeLoadedImageLine(line string) string {
	for _, prefix := range []string{"Loaded image:", "Loaded image ID:"} {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return line
}

func summarizeImportError(err error) string {
	if err == nil {
		return "未知错误"
	}
	if errors.Is(err, ErrImportTooLarge) {
		return "镜像包超过大小限制"
	}
	text := strings.TrimSpace(err.Error())
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "invalid tar header"),
		strings.Contains(lower, "unexpected eof"),
		strings.Contains(lower, "archive/tar"),
		strings.Contains(lower, "invalid manifest"),
		strings.Contains(lower, "gzip"),
		strings.Contains(lower, "bzip2"):
		return "镜像包格式不正确或已损坏，请上传 docker save 导出的 tar 文件"
	default:
		const maxLength = 200
		if len([]rune(text)) > maxLength {
			return string([]rune(text)[:maxLength]) + "…"
		}
		return text
	}
}

// humanSize 用于错误提示中的上限展示。
func humanSize(bytes int64) string {
	if bytes <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(bytes)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", bytes, units[unit])
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}
