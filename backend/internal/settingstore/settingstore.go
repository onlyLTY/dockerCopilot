package settingstore

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"github.com/onlyLTY/dockerCopilot/internal/datadir"
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
	maxHubURLs                 = 20
)

// DefaultHubURLs 官方 Docker Hub 检查更新时的默认加速源（仅 host，按优先级）。
// 与历史 DefaultAcceleratorHostList 保持一致；用户可在设置中增删改。
var DefaultHubURLs = []string{
	"docker.1ms.run",
	"docker.m.daocloud.io",
	"docker.1panel.top",
	"docker.1panel.live",
	"proxy.1panel.live",
	"dockerproxy.1panel.live",
	"docker.1panel.dev",
	"docker.anye.in",
	"hub.rat.dev",
	"docker.amingg.com",
}

var settingsMu sync.Mutex

// Settings 应用级设置，持久化到 {DATA_DIR}/config/appSettings.json（默认 /data）。
type Settings struct {
	UpdateCheckInterval     string   `json:"updateCheckInterval"`
	AutoBackupInterval      string   `json:"autoBackupInterval"`
	LogLevel                string   `json:"logLevel"`
	Retention               int      `json:"retention"`
	IgnoredContainerUpdates []string `json:"ignoredContainerUpdates,omitempty"`
	// HubURLs 官方 Docker Hub（docker.io）镜像检查更新时的加速源列表（仅 host，有序）。
	// 未配置时 GetHubURLs 返回 DefaultHubURLs；用户保存后（含空列表）以持久化值为准。
	HubURLs                 []string `json:"hubUrls,omitempty"`
	HubURLsConfigured       bool     `json:"hubUrlsConfigured,omitempty"`
	GithubProxy             string   `json:"githubProxy,omitempty"`
	HTTPProxy               string   `json:"HTTP_PROXY,omitempty"`
	HTTPSProxy              string   `json:"HTTPS_PROXY,omitempty"`
	NoProxy                 string   `json:"NO_PROXY,omitempty"`
	ProxySettingsConfigured bool     `json:"proxySettingsConfigured,omitempty"`
}

// SettingsPath 设置文件路径。可用 APP_SETTINGS_PATH 覆盖，便于测试；否则 {DATA_DIR}/config/appSettings.json。
func SettingsPath() string {
	if p := os.Getenv("APP_SETTINGS_PATH"); p != "" {
		return p
	}
	return datadir.AppSettingsPath()
}

func load() Settings {
	s := Settings{
		UpdateCheckInterval: defaultUpdateCheckInterval,
		AutoBackupInterval:  defaultAutoBackupInterval,
		LogLevel:            defaultLogLevel,
		Retention:           defaultRetention,
		HubURLs:             append([]string(nil), DefaultHubURLs...),
	}
	content, err := os.ReadFile(SettingsPath())
	if err != nil {
		return s
	}
	var stored Settings
	if err := json.Unmarshal(content, &stored); err != nil {
		return s
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(content, &fields); err == nil {
		_, hasGithubProxy := fields["githubProxy"]
		_, hasHTTPProxy := fields["HTTP_PROXY"]
		_, hasHTTPSProxy := fields["HTTPS_PROXY"]
		_, hasNoProxy := fields["NO_PROXY"]
		_, hasProxyMarker := fields["proxySettingsConfigured"]
		stored.ProxySettingsConfigured = stored.ProxySettingsConfigured || hasGithubProxy || hasHTTPProxy || hasHTTPSProxy || hasNoProxy || hasProxyMarker
		_, hasHubURLs := fields["hubUrls"]
		_, hasHubURLsMarker := fields["hubUrlsConfigured"]
		stored.HubURLsConfigured = stored.HubURLsConfigured || hasHubURLs || hasHubURLsMarker
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
	s.IgnoredContainerUpdates = normalizeIgnoredContainers(stored.IgnoredContainerUpdates)
	if stored.HubURLsConfigured {
		s.HubURLs = normalizeHubURLs(stored.HubURLs)
		s.HubURLsConfigured = true
	}
	s.GithubProxy = stored.GithubProxy
	s.HTTPProxy = stored.HTTPProxy
	s.HTTPSProxy = stored.HTTPSProxy
	s.NoProxy = stored.NoProxy
	s.ProxySettingsConfigured = stored.ProxySettingsConfigured
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

// ===== 容器更新忽略 =====

func normalizeIgnoredContainers(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = filepath.Base(strings.TrimSpace(item))
		if item == "" || item == "." {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

// IsContainerUpdateIgnored 判断指定容器名称是否永久忽略更新提示。
func IsContainerUpdateIgnored(name string) bool {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == "" {
		return false
	}
	for _, ignored := range load().IgnoredContainerUpdates {
		if ignored == name {
			return true
		}
	}
	return false
}

// SetContainerUpdateIgnored 设置或取消指定容器名称的更新忽略状态。
func SetContainerUpdateIgnored(name string, ignored bool) error {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == "" {
		return fmt.Errorf("容器名称不能为空")
	}
	return update(func(s *Settings) error {
		items := normalizeIgnoredContainers(s.IgnoredContainerUpdates)
		found := false
		for _, item := range items {
			if item == name {
				found = true
				break
			}
		}
		if ignored {
			if !found {
				items = append(items, name)
			}
			s.IgnoredContainerUpdates = items
			return nil
		}
		filtered := make([]string, 0, len(items))
		for _, item := range items {
			if item != name {
				filtered = append(filtered, item)
			}
		}
		s.IgnoredContainerUpdates = filtered
		return nil
	})
}

// RenameContainerUpdateIgnore 将容器重命名后的忽略状态迁移到新名称。
func RenameContainerUpdateIgnore(oldName, newName string) error {
	oldName = filepath.Base(strings.TrimSpace(oldName))
	newName = filepath.Base(strings.TrimSpace(newName))
	if oldName == "" || oldName == "." || newName == "" || newName == "." || oldName == newName {
		return nil
	}
	return update(func(s *Settings) error {
		items := normalizeIgnoredContainers(s.IgnoredContainerUpdates)
		found := false
		filtered := make([]string, 0, len(items)+1)
		for _, item := range items {
			if item == oldName {
				found = true
				continue
			}
			filtered = append(filtered, item)
		}
		if found {
			filtered = append(filtered, newName)
		}
		s.IgnoredContainerUpdates = normalizeIgnoredContainers(filtered)
		return nil
	})
}

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
// 说明：应用层保留 warn；写入 go-zero logx 时 warn 会映射为 error（logx 无独立 warn 档）。
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

// ===== Docker Hub 加速源（检查更新） =====

// normalizeHubURLs 清洗加速源列表：去空白、统一为 host、去重并保持顺序。
// 接受纯 host（docker.m.daocloud.io）或带协议的 URL（https://docker.m.daocloud.io/）。
func normalizeHubURLs(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		host, ok := normalizeHubURLEntry(item)
		if !ok {
			continue
		}
		if _, exists := seen[host]; exists {
			continue
		}
		seen[host] = struct{}{}
		result = append(result, host)
	}
	return result
}

func normalizeHubURLEntry(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	for _, r := range raw {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return "", false
		}
	}
	// 允许用户粘贴完整 URL；统一存 host。带 path/query 的地址拒绝（避免 /v2 等被静默丢掉）。
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" {
			return "", false
		}
		if parsed.User != nil {
			return "", false
		}
		if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
			return "", false
		}
		path := strings.Trim(parsed.EscapedPath(), "/")
		if path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return "", false
		}
		raw = parsed.Host
	}
	raw = strings.Trim(raw, "/")
	if raw == "" || strings.ContainsAny(raw, "/?#") {
		return "", false
	}
	// host 基本校验：至少含一个点或为 localhost / 纯 IP 也可；这里只拒绝明显非法字符
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == ':' || r == '[' || r == ']' {
			continue
		}
		return "", false
	}
	return strings.ToLower(raw), true
}

func validHubURLEntry(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if _, ok := normalizeHubURLEntry(raw); !ok {
		return fmt.Errorf("hubUrls 含有无效地址：%s（请填写 host 或 http(s) URL，不要带路径/账号）", raw)
	}
	return nil
}

// GetHubURLs 读取 Docker Hub 加速源列表。
// 用户从未保存过时返回 DefaultHubURLs；保存过（含空列表）返回持久化值。
func GetHubURLs() []string {
	s := load()
	out := make([]string, len(s.HubURLs))
	copy(out, s.HubURLs)
	return out
}

// DefaultHubURLList 返回内置默认加速源的副本（供前端「恢复默认」）。
func DefaultHubURLList() []string {
	out := make([]string, len(DefaultHubURLs))
	copy(out, DefaultHubURLs)
	return out
}

// SetHubURLs 校验并持久化加速源列表。允许空列表（表示不使用加速，仅尝试官方）。
func SetHubURLs(items []string) ([]string, error) {
	if len(items) > maxHubURLs {
		return nil, fmt.Errorf("hubUrls 最多 %d 个", maxHubURLs)
	}
	for _, item := range items {
		if err := validHubURLEntry(item); err != nil {
			return nil, err
		}
	}
	normalized := normalizeHubURLs(items)
	if err := update(func(s *Settings) error {
		s.HubURLs = normalized
		s.HubURLsConfigured = true
		return nil
	}); err != nil {
		return nil, err
	}
	return normalized, nil
}

// ProxySettings 返回应用出站请求使用的代理配置。
type ProxySettings struct {
	GithubProxy string `json:"githubProxy"`
	HTTPProxy   string `json:"HTTP_PROXY"`
	HTTPSProxy  string `json:"HTTPS_PROXY"`
	NoProxy     string `json:"NO_PROXY"`
}

// GetProxySettings 读取代理配置，缺省值为空。
func GetProxySettings() ProxySettings {
	s := load()
	return ProxySettings{
		GithubProxy: s.GithubProxy,
		HTTPProxy:   s.HTTPProxy,
		HTTPSProxy:  s.HTTPSProxy,
		NoProxy:     s.NoProxy,
	}
}

func validProxyURL(value string, name string) error {
	if value == "" {
		return nil
	}
	for _, r := range value {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return fmt.Errorf("%s 不能包含空白或控制字符", name)
		}
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s 必须是包含协议和主机的 URL", name)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" && parsed.Scheme != "socks5" && parsed.Scheme != "socks5h" {
		return fmt.Errorf("%s 使用了不支持的协议", name)
	}
	if parsed.User != nil {
		return fmt.Errorf("%s 不支持在 URL 中包含账号密码", name)
	}
	return nil
}

func validNoProxy(value string) error {
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("NO_PROXY 不能包含控制字符")
		}
	}
	return nil
}

// SetProxySettings 校验并持久化应用出站请求的代理配置。
func SetProxySettings(value ProxySettings) (ProxySettings, error) {
	if err := validProxyURL(value.GithubProxy, "githubProxy"); err != nil {
		return ProxySettings{}, err
	}
	if err := validProxyURL(value.HTTPProxy, "HTTP_PROXY"); err != nil {
		return ProxySettings{}, err
	}
	if err := validProxyURL(value.HTTPSProxy, "HTTPS_PROXY"); err != nil {
		return ProxySettings{}, err
	}
	if err := validNoProxy(value.NoProxy); err != nil {
		return ProxySettings{}, err
	}
	value.GithubProxy = strings.TrimSpace(value.GithubProxy)
	value.HTTPProxy = strings.TrimSpace(value.HTTPProxy)
	value.HTTPSProxy = strings.TrimSpace(value.HTTPSProxy)
	value.NoProxy = strings.TrimSpace(value.NoProxy)
	if err := update(func(s *Settings) error {
		s.GithubProxy = value.GithubProxy
		s.HTTPProxy = value.HTTPProxy
		s.HTTPSProxy = value.HTTPSProxy
		s.NoProxy = value.NoProxy
		s.ProxySettingsConfigured = true
		return nil
	}); err != nil {
		return ProxySettings{}, err
	}
	return value, nil
}

func ApplyProxySettings() error {
	settings := load()
	if !settings.ProxySettingsConfigured {
		return nil
	}
	values := map[string]string{
		"githubProxy": settings.GithubProxy,
		"HTTP_PROXY":  settings.HTTPProxy,
		"HTTPS_PROXY": settings.HTTPSProxy,
		"NO_PROXY":    settings.NoProxy,
	}
	for name, value := range values {
		if err := os.Setenv(name, value); err != nil {
			return err
		}
	}
	return nil
}
