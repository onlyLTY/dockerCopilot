package settingstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// 更新检查频率的预设选项。key 为前端下拉选项值，cron 为对应的 cron 表达式（分 时 日 月 周）。
// off 表示关闭定时检查（仍可手动触发）。
var updateCheckCrons = map[string]string{
	"off": "",
	"30m": "*/30 * * * *",
	"1h":  "30 * * * *",
	"6h":  "30 */6 * * *",
	"12h": "30 */12 * * *",
	"24h": "30 3 * * *",
}

// 自动备份频率的预设选项。off 表示关闭自动备份（仍可手动创建）。
var autoBackupCrons = map[string]string{
	"off":   "",
	"6h":    "0 */6 * * *",
	"12h":   "0 */12 * * *",
	"24h":   "0 4 * * *",
	"week":  "0 4 * * 0",
	"month": "0 4 1 * *",
}

var logLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
}

const (
	defaultUpdateCheckInterval = "1h"
	defaultAutoBackupInterval  = "off"
	defaultLogLevel            = "info"
	defaultRetention           = 10
	minRetention               = 1
	maxRetention               = 100
)

var settingsMu sync.Mutex

// Settings 应用级设置，持久化到 /data/config/appSettings.json
type Settings struct {
	UpdateCheckInterval string `json:"updateCheckInterval"`
	AutoBackupInterval  string `json:"autoBackupInterval"`
	LogLevel            string `json:"logLevel"`
	Retention           int    `json:"retention"`
}

// SettingsPath 设置文件路径。可用 APP_SETTINGS_PATH 覆盖，便于测试。
func SettingsPath() string {
	if p := os.Getenv("APP_SETTINGS_PATH"); p != "" {
		return p
	}
	return "/data/config/appSettings.json"
}

func load() Settings {
	s := Settings{
		UpdateCheckInterval: defaultUpdateCheckInterval,
		AutoBackupInterval:  defaultAutoBackupInterval,
		LogLevel:            defaultLogLevel,
		Retention:           defaultRetention,
	}
	content, err := os.ReadFile(SettingsPath())
	if err != nil {
		return s
	}
	var stored Settings
	if err := json.Unmarshal(content, &stored); err != nil {
		return s
	}
	if ValidUpdateCheckInterval(stored.UpdateCheckInterval) {
		s.UpdateCheckInterval = stored.UpdateCheckInterval
	}
	if ValidAutoBackupInterval(stored.AutoBackupInterval) {
		s.AutoBackupInterval = stored.AutoBackupInterval
	}
	if ValidLogLevel(stored.LogLevel) {
		s.LogLevel = stored.LogLevel
	}
	if stored.Retention >= minRetention && stored.Retention <= maxRetention {
		s.Retention = stored.Retention
	}
	return s
}

func save(s Settings) error {
	if err := os.MkdirAll(filepath.Dir(SettingsPath()), 0755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(SettingsPath(), content, 0644)
}

func update(fn func(*Settings) error) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	s := load()
	if err := fn(&s); err != nil {
		return err
	}
	return save(s)
}

// ===== 更新检查频率 =====

// UpdateCheckOptions 返回可选的更新检查频率（供前端下拉展示）。
func UpdateCheckOptions() []string {
	return []string{"off", "30m", "1h", "6h", "12h", "24h"}
}

// ValidUpdateCheckInterval 判断给定 key 是否为合法预设。
func ValidUpdateCheckInterval(key string) bool {
	_, ok := updateCheckCrons[key]
	return ok
}

// UpdateCheckCron 返回给定预设 key 对应的 cron 表达式；off 或未知返回空串。
func UpdateCheckCron(key string) string {
	return updateCheckCrons[key]
}

// GetUpdateCheckInterval 读取当前更新检查频率预设，缺省返回默认值。
func GetUpdateCheckInterval() string {
	return load().UpdateCheckInterval
}

// SetUpdateCheckInterval 校验并持久化更新检查频率预设。
func SetUpdateCheckInterval(key string) (string, error) {
	if !ValidUpdateCheckInterval(key) {
		return "", fmt.Errorf("无效的更新检查频率：%s", key)
	}
	if err := update(func(s *Settings) error {
		s.UpdateCheckInterval = key
		return nil
	}); err != nil {
		return "", err
	}
	return key, nil
}

// ===== 自动备份频率 =====

// AutoBackupOptions 返回可选的自动备份频率（供前端下拉展示）。
func AutoBackupOptions() []string {
	return []string{"off", "6h", "12h", "24h", "week", "month"}
}

// ValidAutoBackupInterval 判断给定 key 是否为合法预设。
func ValidAutoBackupInterval(key string) bool {
	_, ok := autoBackupCrons[key]
	return ok
}

// AutoBackupCron 返回给定预设 key 对应的 cron 表达式；off 或未知返回空串。
func AutoBackupCron(key string) string {
	return autoBackupCrons[key]
}

// GetAutoBackupInterval 读取当前自动备份频率预设，缺省返回默认值。
func GetAutoBackupInterval() string {
	return load().AutoBackupInterval
}

// SetAutoBackupInterval 校验并持久化自动备份频率预设。
func SetAutoBackupInterval(key string) (string, error) {
	if !ValidAutoBackupInterval(key) {
		return "", fmt.Errorf("无效的自动备份频率：%s", key)
	}
	if err := update(func(s *Settings) error {
		s.AutoBackupInterval = key
		return nil
	}); err != nil {
		return "", err
	}
	return key, nil
}

// ===== 备份保留数量 =====

// ValidRetention 判断备份保留数量是否有效。
func ValidRetention(value int) bool {
	return value >= minRetention && value <= maxRetention
}

// GetRetention 读取当前备份保留数量。
func GetRetention() int {
	return load().Retention
}

// SetRetention 校验并持久化备份保留数量。
func SetRetention(value int) (int, error) {
	if !ValidRetention(value) {
		return 0, fmt.Errorf("备份保留数量必须在 %d-%d 之间", minRetention, maxRetention)
	}
	if err := update(func(s *Settings) error {
		s.Retention = value
		return nil
	}); err != nil {
		return 0, err
	}
	return value, nil
}

// LogLevelOptions 返回可选的日志级别。
func LogLevelOptions() []string {
	return []string{"debug", "info", "warn", "error"}
}

// ValidLogLevel 判断日志级别是否有效。
func ValidLogLevel(level string) bool {
	return logLevels[level]
}

// GetLogLevel 读取当前日志级别。
func GetLogLevel() string {
	return load().LogLevel
}

// SetLogLevel 校验并持久化日志级别。
func SetLogLevel(level string) (string, error) {
	if !ValidLogLevel(level) {
		return "", fmt.Errorf("无效的日志级别：%s", level)
	}
	if err := update(func(s *Settings) error {
		s.LogLevel = level
		return nil
	}); err != nil {
		return "", err
	}
	return level, nil
}
