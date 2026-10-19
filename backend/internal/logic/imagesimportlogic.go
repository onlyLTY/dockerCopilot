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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	dockerMsgType "github.com/docker/docker/pkg/jsonmessage"
	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/datadir"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// ErrImportTooLarge 上传的镜像包超过配置上限。
// handler 把 MaxBytesReader 的错误映射为该哨兵，便于返回明确的业务文案。
var ErrImportTooLarge = errors.New("镜像包超过大小限制")

// errUploadInterrupted 上传未完成（客户端断开或请求被中断）。
var errUploadInterrupted = errors.New("镜像包上传中断")

const (
	importTaskPrefix       = "导入镜像 "
	importSubmitMessage    = "任务已提交"
	importUploadMessage    = "正在上传镜像包"
	importLoadMessage      = "正在导入镜像"
	importDoneMessage      = "导入完成"
	importFailedMessage    = "导入失败"
	importTempFilePrefix   = "image-import-"
	importProgressInterval = 250 * time.Millisecond
)

// taskIDPattern 限制客户端生成的任务编号，避免任意字符串进入进度库。
var taskIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

// ImportInput 已解析的上传流；Reader 直接从网络读取，上传过程中只写入临时文件。
type ImportInput struct {
	Filename string
	TaskID   string
	Reader   io.Reader
	// Total 为 Content-Length；未知时为 0 或负数，进度按不确定态展示。
	Total int64
}

// importSubmitResult 提交成功后返回；实际导入结果写入任务进度。
type importSubmitResult struct {
	TaskID   string `json:"taskID"`
	Filename string `json:"filename"`
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

// SubmitImport 接收 docker save（可 gzip/bzip2 压缩）的镜像包：
// 先把请求体流式写入数据目录下的临时文件，上传完成后立即返回 taskID，
// 再由后台任务执行 docker load，任务页可查看进度并停止任务。
func (l *ImportImagesLogic) SubmitImport(in ImportInput) (*types.Resp, error) {
	resp := &types.Resp{}
	if err := l.svcCtx.RequireDocker(); err != nil {
		return Fail(resp, err, 500, "导入镜像失败")
	}
	if in.Reader == nil {
		return Biz(400, "请选择要导入的镜像包", nil), nil
	}

	taskID := strings.TrimSpace(in.TaskID)
	switch {
	case taskID == "":
		taskID = uuid.NewString()
	case !taskIDPattern.MatchString(taskID):
		return Biz(400, "任务编号格式不正确", nil), nil
	}
	filename := strings.TrimSpace(in.Filename)
	if filename == "" {
		filename = "image.tar"
	}
	filename = truncateRunes(filename, 200)
	name := importTaskPrefix + filename

	// 已存在的任务编号（含已完成记录）不能复用：UpdateProgress 会拒绝覆盖终态。
	if _, exists := l.svcCtx.GetProgress(taskID); exists {
		return Biz(409, "任务编号已存在，请重试", nil), nil
	}
	taskCtx := l.svcCtx.RegisterTask(taskID)
	if taskCtx == nil {
		return Biz(409, "任务编号已被占用或服务正在关闭", nil), nil
	}

	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
		TaskID: taskID, Refresh: true, Name: name,
		Message: importSubmitMessage, Percentage: 10,
		ProgressType: svc.ProgressTypeImageImport, Indeterminate: true,
	})

	tempPath, err := l.saveUpload(taskCtx, taskID, name, in)
	if err != nil {
		l.svcCtx.FinishTask(taskID)
		return l.uploadFailure(taskCtx, taskID, name, err)
	}

	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
		TaskID: taskID, Refresh: true, Name: name,
		Message: importLoadMessage, Percentage: 60,
		ProgressType: svc.ProgressTypeImageImport, Indeterminate: true,
	})

	go runImageImport(taskCtx, l.svcCtx, taskID, name, tempPath)

	return Ok(resp, importSubmitResult{TaskID: taskID, Filename: filename}), nil
}

// saveUpload 把请求体写入临时文件，并按写入字节数更新上传进度。
func (l *ImportImagesLogic) saveUpload(taskCtx context.Context, taskID, name string, in ImportInput) (string, error) {
	if err := os.MkdirAll(datadir.TempDir(), 0755); err != nil {
		return "", fmt.Errorf("无法创建临时目录：%w", err)
	}
	file, err := os.CreateTemp(datadir.TempDir(), importTempFilePrefix+"*.tar")
	if err != nil {
		return "", fmt.Errorf("无法创建临时文件：%w", err)
	}
	tempPath := file.Name()
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(tempPath)
	}

	total := in.Total
	if total < 0 {
		total = 0
	}
	reader := &uploadProgressReader{
		ctx:    taskCtx,
		reader: in.Reader,
		total:  total,
		onProgress: func(current, total int64, percent int, indeterminate bool) {
			l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
				TaskID: taskID, Refresh: true, Name: name,
				Message: importUploadMessage, Percentage: percent,
				ProgressType:  svc.ProgressTypeImageImport,
				Indeterminate: indeterminate,
				Current:       current, Total: total,
				StepPercentage: uploadStepPercentage(current, total),
			})
		},
	}
	if _, err := io.Copy(file, reader); err != nil {
		cleanup()
		// 声明了长度却没读够，通常是客户端断开或上传被取消，而不是镜像包损坏
		if total > 0 && reader.current < total && !isTooLargeError(err) {
			return "", fmt.Errorf("%w：已接收 %s / %s", errUploadInterrupted, humanSize(reader.current), humanSize(total))
		}
		return "", err
	}
	if total > 0 && reader.current < total {
		cleanup()
		return "", fmt.Errorf("%w：已接收 %s / %s", errUploadInterrupted, humanSize(reader.current), humanSize(total))
	}
	if err := file.Sync(); err != nil {
		cleanup()
		return "", fmt.Errorf("写入镜像包失败：%w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("写入镜像包失败：%w", err)
	}
	info, err := os.Stat(tempPath)
	if err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("写入镜像包失败：%w", err)
	}
	if info.Size() == 0 {
		_ = os.Remove(tempPath)
		return "", errors.New("镜像包为空")
	}
	if err := taskCtx.Err(); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}
	return tempPath, nil
}

func (l *ImportImagesLogic) uploadFailure(taskCtx context.Context, taskID, name string, err error) (*types.Resp, error) {
	resp := &types.Resp{}
	if taskCtx.Err() != nil || errors.Is(err, context.Canceled) {
		l.svcCtx.MarkTaskCanceled(taskID, "任务已停止；未完成上传的镜像包已丢弃")
		return Fail(resp, err, 500, "导入已取消")
	}
	if isTooLargeError(err) {
		msg := fmt.Sprintf("镜像包超过大小限制（最大 %s）", humanSize(l.svcCtx.Config.ImageImportLimit()))
		l.failTask(taskID, name, msg)
		return Biz(413, msg, nil), nil
	}
	message := summarizeImportError(err)
	l.Errorf("上传镜像包失败: %v", err)
	l.failTask(taskID, name, message)
	return Biz(500, "导入镜像失败："+message, nil), nil
}

func (l *ImportImagesLogic) failTask(taskID, name, detail string) {
	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
		TaskID: taskID, Refresh: true, Name: name,
		Percentage: 100, Message: importFailedMessage,
		DetailMsg: detail, Failed: true, IsDone: true,
	})
}

// runImageImport 在后台执行 docker load；临时文件无论成败都会删除。
func runImageImport(taskCtx context.Context, svcCtx *svc.ServiceContext, taskID, name, tempPath string) {
	defer os.Remove(tempPath)
	defer svcCtx.FinishTask(taskID)
	defer func() {
		if recovered := recover(); recovered != nil {
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{
				TaskID: taskID, Refresh: true, Name: name, Percentage: 100,
				Message: importFailedMessage, DetailMsg: fmt.Sprintf("导入任务异常：%v", recovered),
				Failed: true, IsDone: true,
			})
		}
	}()

	// 镜像操作槽位被其他任务占用时先提示等待，避免任务看起来卡在 60%
	if !svcCtx.TryAcquireImageOp() {
		svcCtx.UpdateProgress(taskID, svc.TaskProgress{
			TaskID: taskID, Refresh: true, Name: name,
			Message: "等待其他镜像操作结束", Percentage: 60,
			ProgressType: svc.ProgressTypeImageImport, Indeterminate: true,
			StepPercentage: 100,
		})
		if !svcCtx.AcquireImageOp(taskCtx) {
			if taskCtx.Err() != nil {
				svcCtx.MarkTaskCanceled(taskID, "任务已停止；镜像包未导入")
				return
			}
			finishImportFailure(taskCtx, svcCtx, taskID, name, errors.New("镜像操作已停止"))
			return
		}
	}
	defer svcCtx.ReleaseImageOp()

	file, err := os.Open(tempPath)
	if err != nil {
		finishImportFailure(taskCtx, svcCtx, taskID, name, fmt.Errorf("无法读取临时镜像包：%w", err))
		return
	}
	defer file.Close()

	reader, closeReader, err := decompressImageArchive(file)
	if err != nil {
		finishImportFailure(taskCtx, svcCtx, taskID, name, err)
		return
	}
	defer closeReader()

	if err := svcCtx.RequireDocker(); err != nil {
		finishImportFailure(taskCtx, svcCtx, taskID, name, err)
		return
	}
	loadResp, err := svcCtx.DockerClient.ImageLoad(taskCtx, reader)
	if err != nil {
		finishImportFailure(taskCtx, svcCtx, taskID, name, err)
		return
	}
	defer loadResp.Body.Close()

	loaded, err := readImageLoadStream(loadResp.Body)
	if err != nil {
		finishImportFailure(taskCtx, svcCtx, taskID, name, err)
		return
	}
	if len(loaded) == 0 {
		finishImportFailure(taskCtx, svcCtx, taskID, name, errors.New("镜像包中未找到可导入的镜像"))
		return
	}

	detail := fmt.Sprintf("已导入 %d 个镜像", len(loaded))
	for _, ref := range loaded {
		detail += "\n- " + ref
	}
	logx.Infof("镜像导入任务 %s 完成，共 %d 个镜像", taskID, len(loaded))
	svcCtx.UpdateProgress(taskID, svc.TaskProgress{
		TaskID: taskID, Refresh: true, Name: name,
		Percentage: 100, Message: importDoneMessage,
		DetailMsg: detail, Succeeded: true, IsDone: true,
	})
}

func finishImportFailure(taskCtx context.Context, svcCtx *svc.ServiceContext, taskID, name string, err error) {
	if taskCtx.Err() != nil {
		svcCtx.MarkTaskCanceled(taskID, "任务已停止；已完成的镜像导入不会自动回滚")
		return
	}
	message := summarizeImportError(err)
	logx.Errorf("导入镜像任务 %s 失败: %v", taskID, err)
	svcCtx.UpdateProgress(taskID, svc.TaskProgress{
		TaskID: taskID, Refresh: true, Name: name,
		Percentage: 100, Message: importFailedMessage,
		DetailMsg: message, Failed: true, IsDone: true,
	})
}

type uploadProgressReader struct {
	ctx        context.Context
	reader     io.Reader
	current    int64
	total      int64
	lastReport time.Time
	onProgress func(current, total int64, percent int, indeterminate bool)
}

func (r *uploadProgressReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	r.current += int64(n)
	if r.onProgress != nil && (r.lastReport.IsZero() || time.Since(r.lastReport) >= importProgressInterval || err != nil) {
		r.lastReport = time.Now()
		indeterminate := r.total <= 0
		percent := 10
		if !indeterminate {
			percent = 10 + int(r.current*40/r.total)
			if percent > 50 {
				percent = 50
			}
		}
		r.onProgress(r.current, r.total, percent, indeterminate)
	}
	return n, err
}

func uploadStepPercentage(current, total int64) int {
	if total <= 0 {
		return 0
	}
	percent := int(current * 100 / total)
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}

// CleanupStaleImageImports 清理进程异常退出后遗留的导入临时文件。
// 服务启动时没有活动任务，超过 maxAge 的残留文件可以安全删除。
func CleanupStaleImageImports(maxAge time.Duration) {
	entries, err := os.ReadDir(datadir.TempDir())
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), importTempFilePrefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		_ = os.Remove(filepath.Join(datadir.TempDir(), entry.Name()))
	}
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
	if errors.Is(err, errUploadInterrupted) {
		return err.Error()
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

func truncateRunes(value string, maxLength int) string {
	runes := []rune(value)
	if len(runes) <= maxLength {
		return value
	}
	return string(runes[:maxLength])
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
