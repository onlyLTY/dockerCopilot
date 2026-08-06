// Package datadir 统一应用数据根目录（备份、图标、设置、任务进度等）。
// 默认 /data；可用环境变量 DATA_DIR 覆盖，便于本地 go run 与测试。
package datadir

import (
	"os"
	"path/filepath"
	"strings"
)

// Root 返回数据根目录。优先 DATA_DIR，否则 /data。
func Root() string {
	if d := strings.TrimSpace(os.Getenv("DATA_DIR")); d != "" {
		return d
	}
	return "/data"
}

// Join 在数据根下拼接子路径。
func Join(elem ...string) string {
	parts := make([]string, 0, len(elem)+1)
	parts = append(parts, Root())
	parts = append(parts, elem...)
	return filepath.Join(parts...)
}

// ConfigDir 配置目录：{DATA_DIR}/config
func ConfigDir() string { return Join("config") }

// ConfigFile 配置文件：{DATA_DIR}/config/{name}
func ConfigFile(name string) string { return Join("config", name) }

// BackupsDir 备份目录默认路径（仍可被 BACKUP_DIR 覆盖，见 backupstore/pathsafe）。
func BackupsDir() string { return Join("backups") }

// IconDir 自定义图标文件目录。
func IconDir() string { return Join("icon", "icons") }

// IconConfigPath 图标映射配置文件。
func IconConfigPath() string { return Join("icon", "imageLogos.js") }

// TaskProgressPath 任务进度持久化文件。
func TaskProgressPath() string { return Join("config", "taskProgress.json") }

// AppSettingsPath 应用设置文件。
func AppSettingsPath() string { return Join("config", "appSettings.json") }

// LoginAttemptsPath 登录失败/封禁计数持久化文件（重启后仍生效）。
func LoginAttemptsPath() string { return Join("config", "loginAttempts.json") }
