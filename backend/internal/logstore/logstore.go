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
	// 扫描体积上限：按级别过滤时可能需要多读一些文件才能凑满 limit
	maxTotalBytes = 2 * 1024 * 1024
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

// ReadRecent 返回最近的日志条目（最新在前）。
// level 为空或 "all" 时不过滤；为 debug/info/warn/error 时在服务端按级别筛选后再截断，
// 因此 level=error&limit=100 表示「最近最多 100 条 error」，而非「混合 100 条里的 error」。
func ReadRecent(limit int, level string) ([]Entry, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	level = normalizeLevelFilter(level)

	files, err := logFiles(Directory())
	if err != nil {
		return nil, err
	}

	// files 已按 mtime 新→旧；从新文件开始读，收集匹配项直到凑满 limit 或扫过体积上限
	matched := make([]Entry, 0, limit)
	var scanned int64
	for _, file := range files {
		if len(matched) >= limit {
			break
		}
		info, err := os.Stat(file.path)
		if err != nil {
			continue
		}
		// 已扫体积达到上限则停止（避免为凑满 error 读完整盘）
		if scanned > 0 && scanned+info.Size() > maxTotalBytes && len(matched) > 0 {
			break
		}
		scanned += info.Size()
		fileEntries, err := readFile(file.path)
		if err != nil {
			continue
		}
		// 单文件内通常旧→新，反转后与「新优先」一致再筛选
		for i, j := 0, len(fileEntries)-1; i < j; i, j = i+1, j-1 {
			fileEntries[i], fileEntries[j] = fileEntries[j], fileEntries[i]
		}
		for _, e := range fileEntries {
			if level != "" && e.Level != level {
				continue
			}
			matched = append(matched, e)
			if len(matched) >= limit {
				break
			}
		}
	}
	return matched, nil
}

func normalizeLevelFilter(level string) string {
	level = strings.ToLower(strings.TrimSpace(level))
	switch level {
	case "", "all", "*":
		return ""
	case "debug", "info", "warn", "error":
		return level
	case "warning":
		return "warn"
	default:
		return ""
	}
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
	// 新 → 旧
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
	const logSeparator = "--------------------------------"
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
