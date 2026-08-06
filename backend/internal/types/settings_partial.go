package types

// UpdateAppSettingsPartial 聚合写入：字段均可选，只更新非 nil 字段。
// 与 goctl 的 UpdateAppSettingsReq 并存；handler 用本类型做 partial Decode。
type UpdateAppSettingsPartial struct {
	UpdateCheckInterval *string            `json:"updateCheckInterval,optional"`
	AutoBackupInterval  *string            `json:"autoBackupInterval,optional"`
	LogLevel            *string            `json:"logLevel,optional"`
	Retention           *int               `json:"retention,optional"`
	Proxy               *ProxySettingsData `json:"proxy,optional"`
}
