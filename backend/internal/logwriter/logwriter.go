package logwriter

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	rotationDelimiter = "-"
	separator         = "--------------------------------"
	timeFormat        = "2006-01-02T15:04:05.000Z07:00"
)

type Writer struct {
	access *logx.RotateLogger
	error  *logx.RotateLogger
	severe *logx.RotateLogger
	slow   *logx.RotateLogger
	stat   *logx.RotateLogger
}

func New(logDir string, keepDays int, compress bool) (*Writer, error) {
	writer := &Writer{}
	newLogger := func(name string) (*logx.RotateLogger, error) {
		path := filepath.Join(logDir, name)
		return logx.NewLogger(path, logx.DefaultRotateRule(path, rotationDelimiter, keepDays, compress), compress)
	}

	var err error
	if writer.access, err = newLogger("access.log"); err != nil {
		return nil, err
	}
	if writer.error, err = newLogger("error.log"); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if writer.severe, err = newLogger("severe.log"); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if writer.slow, err = newLogger("slow.log"); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if writer.stat, err = newLogger("stat.log"); err != nil {
		_ = writer.Close()
		return nil, err
	}

	return writer, nil
}

func (w *Writer) Alert(v any) {
	w.write(w.error, "ERROR", v)
}

func (w *Writer) Close() error {
	var firstErr error
	for _, logger := range []*logx.RotateLogger{w.access, w.error, w.severe, w.slow, w.stat} {
		if logger == nil {
			continue
		}
		if err := logger.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (w *Writer) Debug(v any, _ ...logx.LogField) {
	w.write(w.access, "DEBUG", v)
}

func (w *Writer) Error(v any, _ ...logx.LogField) {
	w.write(w.error, "ERROR", v)
}

func (w *Writer) Info(v any, _ ...logx.LogField) {
	w.write(w.access, "INFO", v)
}

func (w *Writer) Severe(v any) {
	w.write(w.severe, "ERROR", v)
}

func (w *Writer) Slow(v any, _ ...logx.LogField) {
	w.write(w.slow, "ERROR", v)
}

func (w *Writer) Stack(v any) {
	w.write(w.error, "ERROR", v)
}

func (w *Writer) Stat(v any, _ ...logx.LogField) {
	w.write(w.stat, "INFO", v)
}

func (w *Writer) write(logger *logx.RotateLogger, level string, value any) {
	if logger == nil {
		return
	}

	message := strings.ReplaceAll(fmt.Sprint(value), "\r\n", "\n")
	message = strings.TrimRight(message, "\r\n")
	entry := fmt.Sprintf("【%s】%s\n%s\n%s\n", level, time.Now().Format(timeFormat), message, separator)
	_, _ = logger.Write([]byte(entry))
}

var _ logx.Writer = (*Writer)(nil)
