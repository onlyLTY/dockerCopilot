package logstore

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultLogDir = "./logs"
	defaultLimit  = 100
	maxLimit      = 500
	maxTotalBytes = 1024 * 1024
)

type Entry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

func Directory() string {
	if dir := os.Getenv("LOG_DIR"); dir != "" {
		return dir
	}
	return defaultLogDir
}

func ReadRecent(limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	files, err := logFiles(Directory())
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, limit)
	var totalBytes int64
	for index := len(files) - 1; index >= 0; index-- {
		file := files[index]
		info, err := os.Stat(file.path)
		if err != nil {
			continue
		}
		if totalBytes+info.Size() > maxTotalBytes {
			break
		}
		totalBytes += info.Size()
		fileEntries, err := readFile(file.path)
		if err != nil {
			continue
		}
		entries = append(entries, fileEntries...)
	}
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}

type logFile struct {
	path string
	mod  time.Time
}

func logFiles(dir string) ([]logFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []logFile{}, nil
		}
		return nil, err
	}
	files := make([]logFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isLogFile(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		files = append(files, logFile{path: filepath.Join(dir, entry.Name()), mod: info.ModTime()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.After(files[j].mod) })
	return files, nil
}

func isLogFile(name string) bool {
	name = strings.ToLower(name)
	return strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".log.gz")
}

func readFile(path string) ([]Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var reader io.Reader = file
	var compressed *gzip.Reader
	if strings.HasSuffix(strings.ToLower(path), ".gz") {
		compressed, err = gzip.NewReader(file)
		if err != nil {
			return nil, err
		}
		defer compressed.Close()
		reader = compressed
	}
	entries := make([]Entry, 0)
	scanner := bufio.NewScanner(reader)
	buffer := make([]byte, 64*1024)
	scanner.Buffer(buffer, 256*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		entries = append(entries, parseLine(line))
	}
	return entries, scanner.Err()
}

func parseLine(line string) Entry {
	var value map[string]interface{}
	if json.Unmarshal([]byte(line), &value) == nil {
		timestamp := firstString(value, "@timestamp", "timestamp", "time")
		level := normalizeLevel(firstString(value, "level", "severity"))
		message := firstString(value, "content", "message", "msg")
		if message == "" {
			message = line
		}
		return Entry{Timestamp: timestamp, Level: level, Message: message}

	}
	parts := strings.SplitN(line, "\t", 3)
	if len(parts) == 3 {
		level := normalizeLevel(parts[1])
		return Entry{Timestamp: parts[0], Level: level, Message: parts[2]}
	}
	return Entry{Level: "info", Message: line}
}

func normalizeLevel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "debug", "info", "warn", "error":
		return value
	case "warning":
		return "warn"
	case "panic", "fatal":
		return "error"
	default:
		return "info"
	}
}

func firstString(value map[string]interface{}, keys ...string) string {
	for _, key := range keys {

		if text, ok := value[key].(string); ok {
			return text
		}
	}
	return ""
}
