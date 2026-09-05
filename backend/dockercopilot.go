package main

import (
	"context"
	"flag"
	"fmt"
	"go/types"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/onlyLTY/dockerCopilot/internal/config"
	"github.com/onlyLTY/dockerCopilot/internal/datadir"
	"github.com/onlyLTY/dockerCopilot/internal/errorx"
	"github.com/onlyLTY/dockerCopilot/internal/handler"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
	xhttp "github.com/zeromicro/x/http"
)

var configFile = flag.String("f", "etc/dockerCopilot.yaml", "the config file")

// webDir 前端静态资源目录（运行时从磁盘读取）。可用 FRONTEND_DIR 覆盖，默认 ./web。
var webDir = func() string {
	if d := os.Getenv("FRONTEND_DIR"); d != "" {
		return d
	}
	return "./web"
}()

type UnauthorizedResponse struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data map[string]interface{} `json:"data"`
}

func main() {
	// 本地/未 ldflags 注入时，从 version 文件解析版本；正式包已由 -X 写入则保持不变
	config.ResolveVersion()

	// 先加载配置，再定 DataDir：settingstore / 日志级别都依赖数据根目录。
	flag.Parse()
	var c config.Config
	err := conf.Load(*configFile, &c, conf.UseEnv())
	if err != nil {
		logx.Errorf("无法加载配置文件出错: %v", err)
		logx.Errorf("请确认 secretKey（Auth.AccessSecret）已设置：至少 %d 位且不能为纯数字", config.MinAccessSecretLen)
		os.Exit(1)
	}
	if err := config.ValidateAccessSecret(c.Auth.AccessSecret); err != nil {
		logx.Errorf("secretKey 不符合要求: %v", err)
		logx.Errorf("请设置环境变量 secretKey 或配置 Auth.AccessSecret：至少 %d 位且不能为纯数字", config.MinAccessSecretLen)
		os.Exit(1)
	}
	// DATA_DIR 环境变量仍优先；未设时用 yaml DataDir；都空则 /data
	datadir.SetFromConfig(c.DataDir)

	if err := settingstore.ApplyProxySettings(); err != nil {
		logx.Errorf("应用代理设置失败: %v", err)
	}
	logDir := os.Getenv("LOG_DIR")
	if logDir == "" {
		logDir = "./logs"
	}
	if err := SetupLog(logDir, settingstore.GetLogLevel()); err != nil {
		logx.Errorf("failed to setup log: %v", err)
		os.Exit(1)
	}
	serverOptions := []rest.RunOption{rest.WithUnauthorizedCallback(
		func(w http.ResponseWriter, r *http.Request, err error) {
			response := UnauthorizedResponse{
				Code: http.StatusUnauthorized, // 401
				Msg:  "未授权",
				Data: map[string]interface{}{},
			}
			httpx.WriteJson(w, http.StatusUnauthorized, response)
		}),
		rest.WithNotFoundHandler(newFrontendHandler()),
	}
	if len(c.CorsOrigins) > 0 {
		serverOptions = append(serverOptions, rest.WithCors(c.CorsOrigins...))
	}
	server := rest.MustNewServer(c.RestConf, serverOptions...)
	defer server.Stop()
	ctx := svc.NewServiceContext(c)
	// go-zero 在 SIGTERM/SIGINT 时先 WrapUp 再 Shutdown；Linux 容器内生效。
	// Windows 本地 go run 的 polyfill 不会自动触发，进程退出前仍靠 defer Close。
	proc.AddWrapUpListener(func() {
		ctx.FlushProgress()
	})
	proc.AddShutdownListener(func() {
		ctx.Close()
	})
	defer ctx.Close()

	// 自动初始化数据目录（图标/备份/配置）；根目录可由 DATA_DIR 覆盖
	if err := os.MkdirAll(datadir.IconDir(), 0755); err != nil {
		logx.Errorf("Failed to create icon directory: %v", err)
	}
	if err := os.MkdirAll(datadir.BackupsDir(), 0755); err != nil {
		logx.Errorf("Failed to create backup directory: %v", err)
	}
	if err := os.MkdirAll(datadir.ConfigDir(), 0755); err != nil {
		logx.Errorf("Failed to create config directory: %v", err)
	}

	imageLogosPath := datadir.IconConfigPath()
	if _, err := os.Stat(imageLogosPath); os.IsNotExist(err) {
		defaultConfig := []byte(`// 自定义镜像logo配置
export const customImageLogos = {
};
`)
		if err := os.WriteFile(imageLogosPath, defaultConfig, 0644); err != nil {
			logx.Errorf("Failed to create default imageLogos.js: %v", err)
		}
	}

	list, err := utiles.GetImagesList(ctx)
	if err != nil {
		logx.Errorf("获取镜像列表出错: %v", err)
		list = nil
	}
	if list != nil {
		go ctx.HubImageInfo.CheckUpdate(list)
	}
	// 容器 CPU/内存后台采样：容器列表接口直接读缓存返回，不再同步等 Docker 的 1-2 秒/容器 stats 采样
	go utiles.StartStatsCollector(ctx)
	// 更新检查定时任务：频率由 settingstore 持久化配置，可在设置页动态调整。
	updateCheckJob := func(taskCtx context.Context) {
		list, err := utiles.GetImagesListWithContext(taskCtx, ctx)
		if err != nil {
			logx.Errorf("定时获取镜像列表出错: %v", err)
			return
		}
		_ = ctx.HubImageInfo.CheckUpdateWithProgressContext(taskCtx, list, nil)
	}
	// 无论开启还是关闭都注入 job 并初始化调度器：spec 为空时只初始化不添加任务，
	// 便于后续在设置页开启时直接重新调度。
	interval := settingstore.GetUpdateCheckInterval()
	if err := ctx.StartUpdateCron(settingstore.UpdateCheckCron(interval), updateCheckJob); err != nil {
		logx.Errorf("启动更新检查定时任务出错: %v", err)
	}

	// 自动备份定时任务：同时创建 JSON 与 YAML 备份，频率由 settingstore 配置，可在设置页动态调整。
	backupJob := func(taskCtx context.Context) {
		if err := utiles.BackupContainerWithContext(taskCtx, ctx); err != nil {
			logx.Errorf("定时备份（JSON）出错: %v", err)
		}
		if err := utiles.Backup2ComposeWithContext(taskCtx, ctx); err != nil {
			logx.Errorf("定时备份（YAML）出错: %v", err)
		}
	}
	backupInterval := settingstore.GetAutoBackupInterval()
	if err := ctx.StartBackupCron(settingstore.AutoBackupCron(backupInterval), backupJob); err != nil {
		logx.Errorf("启动自动备份定时任务出错: %v", err)
	}
	// 统一错误出口：只认 errorx.CodeError；其它错误只记日志，不回传内部细节。
	httpx.SetErrorHandler(func(err error) (int, any) {
		if ce, ok := errorx.AsCodeError(err); ok {
			return http.StatusOK, xhttp.BaseResponse[types.Nil]{
				Code: ce.Code,
				Msg:  ce.Msg,
			}
		}
		logx.Errorf("unhandled error: %v", err)
		return http.StatusOK, xhttp.BaseResponse[types.Nil]{
			Code: errorx.CodeInternalUnhandled,
			Msg:  "服务内部错误",
		}
	})
	handler.RegisterHandlers(server, ctx)
	RegisterHandlers(server, ctx)
	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	logx.Info("程序版本" + config.Version)
	server.Start()
}

func RegisterHandlers(engine *rest.Server, serverCtx *svc.ServiceContext) {
	// 存活+依赖探测：不鉴权，供 compose healthcheck / 编排使用
	engine.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/healthz",
				Handler: healthzHandler(serverCtx),
			},
		},
	)

	// 自定义上传图标：从 DATA_DIR/icon/icons 提供
	// GET / 已由 handler.RegisterHandlers 中的 webindexHandler 注册，勿重复添加
	iconFileServer := http.StripPrefix("/src/config/image/", http.FileServer(http.Dir(datadir.IconDir())))
	engine.AddRoutes(
		[]rest.Route{
			{
				Method: http.MethodGet,
				Path:   "/src/config/image/:file",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					iconFileServer.ServeHTTP(w, r)
				},
			},
		},
	)

	// 前端静态资源（含 /manager 下任意深度的子目录）由 NotFoundHandler 兜底。
}

// healthzHandler 返回进程存活与 Docker 引擎连通性。
// 200：进程正常且能 Ping 到 Docker；503：进程在但 Docker 不可用。
func healthzHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK
		body := map[string]interface{}{
			"status":  "ok",
			"version": config.Version,
			"docker":  "ok",
		}
		if serverCtx == nil || serverCtx.DockerClient == nil {
			status = http.StatusServiceUnavailable
			body["status"] = "degraded"
			body["docker"] = "unavailable"
		} else {
			pingCtx := r.Context()
			if _, err := serverCtx.DockerClient.Ping(pingCtx); err != nil {
				status = http.StatusServiceUnavailable
				body["status"] = "degraded"
				body["docker"] = "unreachable"
			}
		}
		httpx.WriteJson(w, status, body)
	}
}

// hashedAssetRe 匹配 Angular 构建产物中带内容哈希的 js/css 文件名（如 main-V246GCF6.js）。
var hashedAssetRe = regexp.MustCompile(`-[A-Za-z0-9_-]{8}\.(?:js|css)$`)

// newFrontendHandler 提供 Angular 前端（运行时从 webDir 磁盘目录读取）。
// 命中真实静态文件时按原样返回（带正确的 Content-Type），
// 未命中时回退到 index.html 以支持前端路由（SPA）。
func newFrontendHandler() http.Handler {
	fileServer := http.FileServer(http.Dir(webDir))
	indexPath := filepath.Join(webDir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 去掉 /manager 前缀，使子目录资源可按任意深度访问
		trimmed := strings.TrimPrefix(r.URL.Path, "/manager")
		cleaned := strings.TrimPrefix(path.Clean("/"+trimmed), "/")

		if cleaned != "" {
			// 防目录穿越：清洗后的相对路径拼到 webDir 下再判断
			fp := filepath.Join(webDir, filepath.FromSlash(cleaned))
			if info, err := os.Stat(fp); err == nil && !info.IsDir() {
				r2 := r.Clone(r.Context())
				r2.URL.Path = "/" + cleaned
				// 带内容哈希的构建产物可永久缓存；其余资源短缓存，index.html 回源验证保证发版即生效
				switch {
				case hashedAssetRe.MatchString(cleaned):
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				case strings.HasPrefix(cleaned, "assets/"):
					w.Header().Set("Cache-Control", "public, max-age=3600")
				}
				fileServer.ServeHTTP(w, r2)
				return
			}
		}

		// 未命中静态文件：回退到 index.html
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, indexPath)
	})
}

// logxConfigLevel 将应用日志级别映射到 go-zero logx 档位。
// go-zero 仅有 debug/info/error/severe，无独立 warn：用户选 warn 时按 error 写入（比 info 更少）。
func logxConfigLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug", "info", "error", "severe":
		return strings.ToLower(strings.TrimSpace(level))
	case "warn", "warning":
		return "error"
	default:
		return "info"
	}
}

func ensureLogDirectory(logDir string) error {
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		return os.MkdirAll(logDir, 0755)
	}
	return nil
}

// SetupLog 使用 go-zero 原生 LogConf 初始化文件日志（按天滚动）。
func SetupLog(logDir, level string) error {
	if !settingstore.ValidLogLevel(level) {
		level = "info"
	}
	if err := ensureLogDirectory(logDir); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// go-zero 默认 Rotation=daily：按天滚动文件名，KeepDays 天后删除旧文件；Compress 压缩历史。
	logConf := logx.LogConf{
		ServiceName: "dockerCopilot",
		Mode:        "file",
		Path:        logDir,
		Level:       logxConfigLevel(level),
		KeepDays:    7,
		Compress:    true,
		Rotation:    "daily",
	}

	return logx.SetUp(logConf)
}
