package main

import (
	"flag"
	"fmt"
	"go/types"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/onlyLTY/dockerCopilot/internal/config"
	"github.com/onlyLTY/dockerCopilot/internal/handler"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/x/errors"
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
	logDir := "./logs"
	ErrSetupLog := SetupLog(logDir)
	if ErrSetupLog != nil {
		logx.Errorf("failed to setup log: %v", ErrSetupLog)
		os.Exit(1)
	}
	logx.SetLevel(logx.InfoLevel)

	flag.Parse()
	var c config.Config
	err := conf.Load(*configFile, &c, conf.UseEnv())
	if err != nil {
		logx.Errorf("无法加载配置文件出错: %v", err)
		logx.Errorf("请确认secretKey设置正确，要求非纯数字且大于八位")
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

	// Ensure data directory and config exist (Auto-init)
	dataDir := "/data/icon/icons"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logx.Errorf("Failed to create data directory: %v", err)
	}
	if err := os.MkdirAll("/data/backups", 0755); err != nil {
		logx.Errorf("Failed to create backup directory: %v", err)
	}

	imageLogosPath := "/data/icon/imageLogos.js"
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
	// 更新检查定时任务：频率由 settingstore 持久化配置，可在设置页动态调整。
	updateCheckJob := func() {
		list, err := utiles.GetImagesList(ctx)
		if err != nil {
			logx.Errorf("定时获取镜像列表出错: %v", err)
			return
		}
		ctx.HubImageInfo.CheckUpdate(list)
	}
	// 无论开启还是关闭都注入 job 并初始化调度器：spec 为空时只初始化不添加任务，
	// 便于后续在设置页开启时直接重新调度。
	interval := settingstore.GetUpdateCheckInterval()
	if err := ctx.StartUpdateCron(settingstore.UpdateCheckCron(interval), updateCheckJob); err != nil {
		logx.Errorf("启动更新检查定时任务出错: %v", err)
	}

	// 自动备份定时任务：同时创建 JSON 与 YAML 备份，频率由 settingstore 配置，可在设置页动态调整。
	backupJob := func() {
		if err := utiles.BackupContainer(ctx); err != nil {
			logx.Errorf("定时备份（JSON）出错: %v", err)
		}
		if err := utiles.Backup2Compose(ctx); err != nil {
			logx.Errorf("定时备份（YAML）出错: %v", err)
		}
	}
	backupInterval := settingstore.GetAutoBackupInterval()
	if err := ctx.StartBackupCron(settingstore.AutoBackupCron(backupInterval), backupJob); err != nil {
		logx.Errorf("启动自动备份定时任务出错: %v", err)
	}
	httpx.SetErrorHandler(func(err error) (int, any) {
		switch e := err.(type) {
		case *errors.CodeMsg:
			return http.StatusOK, xhttp.BaseResponse[types.Nil]{
				Code: e.Code,
				Msg:  e.Msg,
			}
		default:
			return http.StatusOK, xhttp.BaseResponse[types.Nil]{
				Code: 50000,
				Msg:  err.Error(),
			}
		}
	})
	handler.RegisterHandlers(server, ctx)
	RegisterHandlers(server)
	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	logx.Info("程序版本" + config.Version)
	server.Start()
}
func RegisterHandlers(engine *rest.Server) {
	// 自定义上传图标：从 /data/icon/icons 目录提供
	iconFileServer := http.StripPrefix("/src/config/image/", http.FileServer(http.Dir("/data/icon/icons")))
	engine.AddRoutes([]rest.Route{{
		Method: http.MethodGet,
		Path:   "/",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/manager", http.StatusMovedPermanently)
		},
	}})

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

	// 前端静态资源（含 /manager 下任意深度的子目录，如 assets/icons、assets/imageIcons）
	// 统一交由 NotFoundHandler 中的前端处理器兜底，支持任意目录深度与前端路由回退。
}

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
				// 命中真实文件，交给标准 FileServer（自动处理 Content-Type/缓存/Range）
				r2 := r.Clone(r.Context())
				r2.URL.Path = "/" + cleaned
				fileServer.ServeHTTP(w, r2)
				return
			}
		}

		// 未命中静态文件：回退到 index.html
		http.ServeFile(w, r, indexPath)
	})
}

// 检查并创建日志目录
func ensureLogDirectory(logDir string) error {
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		return os.MkdirAll(logDir, 0755) // 创建目录并设置权限
	}
	return nil
}

// SetupLog 初始化日志设置
func SetupLog(logDir string) error {
	// 检查日志目录是否存在
	if err := ensureLogDirectory(logDir); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	logConf := logx.LogConf{
		Path:     logDir,
		Level:    "info",
		KeepDays: 7,
		Compress: true,
		Mode:     "file",
	}
	logx.MustSetup(logConf)
	logx.AddWriter(logx.NewWriter(os.Stdout))
	return nil
}
