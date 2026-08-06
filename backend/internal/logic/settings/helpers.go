package settings

import (
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/zeromicro/go-zero/core/logx"
)

// AppSettingsData 聚合设置快照。
type AppSettingsData struct {
	UpdateCheck    IntervalSettings           `json:"updateCheck"`
	AutoBackup     IntervalSettings           `json:"autoBackup"`
	LogLevel       LevelSettings              `json:"logLevel"`
	Retention      int                        `json:"retention"`
	HubURLs        []string                   `json:"hubUrls"`
	DefaultHubURLs []string                   `json:"defaultHubUrls"`
	Proxy          settingstore.ProxySettings `json:"proxy"`
}

type IntervalSettings struct {
	Interval string   `json:"interval"`
	Options  []string `json:"options"`
}

type LevelSettings struct {
	Level   string   `json:"level"`
	Options []string `json:"options"`
}

func SnapshotAppSettings() AppSettingsData {
	return AppSettingsData{
		UpdateCheck: IntervalSettings{
			Interval: settingstore.GetUpdateCheckInterval(),
			Options:  settingstore.UpdateCheckOptions(),
		},
		AutoBackup: IntervalSettings{
			Interval: settingstore.GetAutoBackupInterval(),
			Options:  settingstore.AutoBackupOptions(),
		},
		LogLevel: LevelSettings{
			Level:   settingstore.GetLogLevel(),
			Options: settingstore.LogLevelOptions(),
		},
		Retention:      settingstore.GetRetention(),
		HubURLs:        settingstore.GetHubURLs(),
		DefaultHubURLs: settingstore.DefaultHubURLList(),
		Proxy:          settingstore.GetProxySettings(),
	}
}

// LogLevel 将设置项映射为 go-zero 日志级别。
func LogLevel(level string) uint32 {
	switch level {
	case "debug":
		return logx.DebugLevel
	case "error":
		return logx.ErrorLevel
	default:
		return logx.InfoLevel
	}
}
