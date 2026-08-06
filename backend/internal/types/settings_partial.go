package types

// UpdateAppSettingsPartial 聚合写入：字段均可选，只更新非 nil 字段。
// 与 goctl 的 UpdateAppSettingsReq 并存；handler 用本类型做 partial Decode。
type UpdateAppSettingsPartial struct {
	UpdateCheckInterval *string `json:"updateCheckInterval,optional"`
	AutoBackupInterval  *string `json:"autoBackupInterval,optional"`
	LogLevel            *string `json:"logLevel,optional"`
	Retention           *int    `json:"retention,optional"`
	// PullTimeoutSec 拉取镜像超时（秒），0 表示未配置。
	PullTimeoutSec *int               `json:"pullTimeoutSec,optional"`
	HubURLs        *[]string          `json:"hubUrls,optional"`
	Proxy          *ProxySettingsData `json:"proxy,optional"`
}
