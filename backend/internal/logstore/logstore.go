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
	logSeparator  = "--------------------------------"
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
	return readEntries(reader)
}

func readEntries(reader io.Reader) ([]Entry, error) {
	entries := make([]Entry, 0)
	scanner := bufio.NewScanner(reader)
	buffer := make([]byte, 64*1024)
	scanner.Buffer(buffer, 256*1024)
	var block []string
	flush := func() {
		if len(block) == 0 {
			return
		}
		if entry, ok := parseFormattedBlock(block); ok {
			entries = append(entries, entry)
		} else {
			for _, line := range block {
				line = strings.TrimSpace(line)
				if line != "" {
					entries = append(entries, parseLine(line))
				}
			}
		}
		block = block[:0]
	}
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if strings.TrimSpace(line) == logSeparator {
			flush()
			continue
		}
		if strings.TrimSpace(line) == "" && len(block) == 0 {
			continue
		}
		block = append(block, line)
	}
	flush()
	return entries, scanner.Err()
}

func parseFormattedBlock(lines []string) (Entry, bool) {
	if len(lines) < 1 {
		return Entry{}, false
	}
	first := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(first, "【") {
		return Entry{}, false
	}
	first = strings.TrimPrefix(first, "【")
	endLevel := strings.Index(first, "】")
	if endLevel <= 0 || endLevel+1 >= len(first) {
		return Entry{}, false
	}
	level := normalizeLevel(first[:endLevel])
	timestamp := strings.TrimSpace(first[endLevel+len("】"):])
	if timestamp == "" {
		return Entry{}, false
	}
	return Entry{Timestamp: timestamp, Level: level, Message: strings.Join(lines[1:], "\n")}, true
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
