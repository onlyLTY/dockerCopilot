package settingstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	"off":  "",
	"6h":   "0 */6 * * *",
	"12h":  "0 */12 * * *",
	"24h":  "0 4 * * *",
	"week": "0 4 * * 0",
}

const (
	defaultUpdateCheckInterval = "1h"
	defaultAutoBackupInterval  = "off"
)

// Settings 应用级设置，持久化到 /data/config/appSettings.json
type Settings struct {
	UpdateCheckInterval string `json:"updateCheckInterval"`
	AutoBackupInterval  string `json:"autoBackupInterval"`
}

// SettingsPath 设置文件路径。可用 APP_SETTINGS_PATH 覆盖，便于测试。
func SettingsPath() string {
	if p := os.Getenv("APP_SETTINGS_PATH"); p != "" {
		return p
	}
	return "/data/config/appSettings.json"
}

// load 读取全部设置，文件缺失或损坏时返回带默认值的设置。
func load() Settings {
	s := Settings{UpdateCheckInterval: defaultUpdateCheckInterval, AutoBackupInterval: defaultAutoBackupInterval}
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
	return s
}

// save 以读-改-写回的方式持久化，避免只写单字段时覆盖其他设置。
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
	s := load()
	s.UpdateCheckInterval = key
	if err := save(s); err != nil {
		return "", err
	}
	return key, nil
}

// ===== 自动备份频率 =====

// AutoBackupOptions 返回可选的自动备份频率（供前端下拉展示）。
func AutoBackupOptions() []string {
	return []string{"off", "6h", "12h", "24h", "week"}
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
	s := load()
	s.AutoBackupInterval = key
	if err := save(s); err != nil {
		return "", err
	}
	return key, nil
}
