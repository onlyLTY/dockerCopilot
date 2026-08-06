package types

// UpdateAppSettingsPartial 聚合写入：字段均可选，只更新非 nil 字段。
// 与 goctl 的 UpdateAppSettingsReq 并存；handler 用本类型做 partial Decode。
type UpdateAppSettingsPartial struct {
	UpdateCheckInterval *string `json:"updateCheckInterval,optional"`
	AutoBackupInterval  *string `json:"autoBackupInterval,optional"`
	LogLevel            *string `json:"logLevel,optional"`
	Retention           *int    `json:"retention,optional"`
	// HubURLs 非 nil 时更新 Docker Hub 加速源列表（可为空数组，表示清空加速源）。
	HubURLs *[]string          `json:"hubUrls,optional"`
	Proxy   *ProxySettingsData `json:"proxy,optional"`
}
