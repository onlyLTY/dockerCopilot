package settings

import (
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/zeromicro/go-zero/core/logx"
)

// AppSettingsData 聚合设置快照。
type AppSettingsData struct {
	UpdateCheck IntervalSettings `json:"updateCheck"`
	AutoBackup  IntervalSettings `json:"autoBackup"`
	LogLevel    LevelSettings    `json:"logLevel"`
	Retention   int              `json:"retention"`
	// PullTimeoutSec 拉取镜像超时（秒），0 表示未配置。
	PullTimeoutSec             int                           `json:"pullTimeoutSec"`
	HubURLs                    []string                      `json:"hubUrls"`
	DefaultHubURLs             []string                      `json:"defaultHubUrls"`
	Proxy                      settingstore.ProxySettings    `json:"proxy"`
	DaemonProxyDraft           settingstore.DaemonProxyDraft `json:"daemonProxyDraft"`
	DaemonProxyDraftConfigured bool                          `json:"daemonProxyDraftConfigured"`
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
	s := settingstore.Snapshot()
	return AppSettingsData{
		UpdateCheck:                IntervalSettings{Interval: s.UpdateCheckInterval, Options: settingstore.UpdateCheckOptions()},
		AutoBackup:                 IntervalSettings{Interval: s.AutoBackupInterval, Options: settingstore.AutoBackupOptions()},
		LogLevel:                   LevelSettings{Level: s.LogLevel, Options: settingstore.LogLevelOptions()},
		Retention:                  s.Retention,
		PullTimeoutSec:             s.PullTimeoutSec,
		HubURLs:                    append([]string(nil), s.HubURLs...),
		DefaultHubURLs:             settingstore.DefaultHubURLList(),
		Proxy:                      settingstore.ProxySettings{GithubProxy: s.GithubProxy, HTTPProxy: s.HTTPProxy, HTTPSProxy: s.HTTPSProxy, NoProxy: s.NoProxy},
		DaemonProxyDraft:           s.DaemonProxyDraft,
		DaemonProxyDraftConfigured: s.DaemonProxyConfigured,
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
