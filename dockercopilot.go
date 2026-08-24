package main

import (
	"context"
	"crypto/tls"
	"embed"
	"flag"
	"fmt"
	"go/types"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/config"
	"github.com/onlyLTY/dockerCopilot/internal/handler"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/x/errors"
	xhttp "github.com/zeromicro/x/http"
)

//go:embed front/*
var embeddedFront embed.FS

var (
	configFile      = flag.String("f", "etc/dockerCopilot.yaml", "the config file")
	healthCheckOnly = flag.Bool("healthcheck", false, "check whether the HTTP server port is accepting connections")
)

type UnauthorizedResponse struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data map[string]interface{} `json:"data"`
}

func main() {
	flag.Parse()
	if *healthCheckOnly {
		if err := checkTCPHealth("127.0.0.1:12712", 2*time.Second); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	logDir := "./logs"
	ErrSetupLog := SetupLog(logDir)
	if ErrSetupLog != nil {
		logx.Errorf("failed to setup log: %v", ErrSetupLog)
		os.Exit(1)
	}
	logx.SetLevel(logx.InfoLevel)

	var c config.Config
	err := conf.Load(*configFile, &c, conf.UseEnv())
	if err != nil {
		logx.Errorf("无法加载配置文件出错: %v", err)
		logx.Errorf("请确认 secretKey 环境变量已设置且配置文件格式正确")
		os.Exit(1)
	}
	for _, warning := range runtimeSecurityWarnings(c) {
		logx.Debugf("安全提示（不阻止启动）: %s", warning)
	}
	serverOptions := []rest.RunOption{rest.WithUnauthorizedCallback(
		func(w http.ResponseWriter, r *http.Request, err error) {
			response := UnauthorizedResponse{
				Code: http.StatusUnauthorized, // 401
				Msg:  "未授权",
				Data: map[string]interface{}{},
			}
			httpx.WriteJson(w, http.StatusUnauthorized, response)
		})}
	if rawOrigins := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")); rawOrigins != "" {
		origins := strings.FieldsFunc(rawOrigins, func(r rune) bool { return r == ',' })
		serverOptions = append(serverOptions, rest.WithCors(origins...))
	}
	certFile := strings.TrimSpace(os.Getenv("TLS_CERT_FILE"))
	keyFile := strings.TrimSpace(os.Getenv("TLS_KEY_FILE"))
	if certFile != "" || keyFile != "" {
		if certFile == "" || keyFile == "" {
			logx.Error("TLS_CERT_FILE 和 TLS_KEY_FILE 必须同时配置")
			os.Exit(1)
		}
		certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			logx.Errorf("加载 TLS 证书失败: %v", err)
			os.Exit(1)
		}
		serverOptions = append(serverOptions, rest.WithTLSConfig(&tls.Config{
			MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{certificate},
		}))
	}
	server := rest.MustNewServer(c.RestConf, serverOptions...)
	server.Use(securityHeaders)
	defer server.Stop()
	ctx := svc.NewServiceContext(c)
	if ctx.DockerClient != nil {
		defer ctx.DockerClient.Close()
	}
	imageCheckContext, cancelImageChecks := context.WithCancel(context.Background())
	defer cancelImageChecks()

	// Ensure data directory and config exist (Auto-init)
	dataDir := "/data/config/image"
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		logx.Errorf("Failed to create data directory: %v", err)
		os.Exit(1)
	}

	imageLogosPath := "/data/config/imageLogos.js"
	if _, err := os.Stat(imageLogosPath); os.IsNotExist(err) {
		defaultConfig := []byte(`// 自定义镜像logo配置
export const customImageLogos = {
};
`)
		if err := os.WriteFile(imageLogosPath, defaultConfig, 0600); err != nil {
			logx.Errorf("Failed to create default imageLogos.js: %v", err)
			os.Exit(1)
		}
	} else if err != nil {
		logx.Errorf("Failed to inspect imageLogos.js: %v", err)
		os.Exit(1)
	}

	if list, err := utiles.GetImagesList(ctx); err != nil {
		logx.Errorf("首次获取镜像列表失败，将在定时任务中重试: %v", err)
	} else {
		go ctx.HubImageInfo.CheckUpdate(imageCheckContext, ctx.DockerClient, list)
	}
	corndanmu := cron.New(cron.WithParser(cron.NewParser(
		cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow,
	)), cron.WithChain(cron.Recover(cron.DefaultLogger)))
	_, err = corndanmu.AddFunc("30 * * * *", func() {
		ctx.CleanupProgress(time.Hour)
		list, err := utiles.GetImagesList(ctx)
		if err != nil {
			logx.Errorf("定时获取镜像列表失败: %v", err)
			return
		}
		ctx.HubImageInfo.CheckUpdate(imageCheckContext, ctx.DockerClient, list)
	})
	if err != nil {
		logx.Errorf("添加定时任务失败: %v", err)
	}
	corndanmu.Start()
	defer corndanmu.Stop()
	httpx.SetErrorHandler(func(err error) (int, any) {
		switch e := err.(type) {
		case *errors.CodeMsg:
			return http.StatusBadRequest, xhttp.BaseResponse[types.Nil]{
				Code: e.Code,
				Msg:  e.Msg,
			}
		default:
			logx.Errorf("未处理的 HTTP 错误: %v", err)
			return http.StatusInternalServerError, xhttp.BaseResponse[types.Nil]{
				Code: http.StatusInternalServerError,
				Msg:  "内部服务器错误",
			}
		}
	})
	handler.RegisterHandlers(server, ctx)
	RegisterHandlers(server)
	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	logx.Info("程序版本" + config.Version)
	server.Start()
}

func checkTCPHealth(address string, timeout time.Duration) error {
	connection, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return fmt.Errorf("health check failed for %s: %w", address, err)
	}
	return connection.Close()
}

func runtimeSecurityWarnings(c config.Config) []string {
	warnings := make([]string, 0, 3)
	secret := c.Auth.AccessSecret
	if len(secret) < 32 {
		warnings = append(warnings, "secretKey 少于 32 个字符，建议使用更强的随机密码")
	}
	allDigits := true
	for _, character := range secret {
		if character < '0' || character > '9' {
			allDigits = false
			break
		}
	}
	if secret != "" && allDigits {
		warnings = append(warnings, "secretKey 为纯数字，建议使用包含字母和符号的密码")
	}
	backupSecret := os.Getenv("BACKUP_ENCRYPTION_KEY")
	if backupSecret != "" && len(backupSecret) < 32 {
		warnings = append(warnings, "BACKUP_ENCRYPTION_KEY 少于 32 个字符，建议使用更强的随机密钥")
	}
	return warnings
}

func securityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; connect-src 'self' https://api.github.com; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'; worker-src 'self'")
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		header.Set("Cross-Origin-Opener-Policy", "same-origin")
		header.Set("Cross-Origin-Resource-Policy", "same-origin")
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" {
			header.Set("Cache-Control", "no-store")
			header.Set("Pragma", "no-cache")
		}
		if r.TLS != nil {
			header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next(w, r)
	}
}
func RegisterHandlers(engine *rest.Server) {
	frontFS, err := fs.Sub(embeddedFront, "front")
	if err != nil {
		log.Fatal(err)
	}

	frontFileServer := http.StripPrefix("/manager", http.FileServer(http.FS(frontFS)))

	assetsHandler := http.FileServer(http.FS(frontFS))

	// Serve custom icons
	iconFileServer := http.StripPrefix("/src/config/image/", http.FileServer(http.Dir("/data/config/image")))
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

	engine.AddRoutes(
		[]rest.Route{
			{
				Method: http.MethodGet,
				Path:   "/manager",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					frontFileServer.ServeHTTP(w, r)
				},
			},
			{
				Method: http.MethodGet,
				Path:   "/manager/:path",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					frontFileServer.ServeHTTP(w, r)
				},
			},
			{
				Method: http.MethodGet,
				Path:   "/manager/assets/:path",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					frontFileServer.ServeHTTP(w, r)
				},
			},
			{
				Method: http.MethodGet,
				Path:   "/assets/:path",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					assetsHandler.ServeHTTP(w, r)
				},
			},
		},
	)
}

// 检查并创建日志目录
func ensureLogDirectory(logDir string) error {
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return err
	}
	// #nosec G302 -- directories require execute permission; 0700 is owner-only.
	return os.Chmod(logDir, 0o700)
}

// SetupLog 初始化日志设置
func SetupLog(logDir string) error {
	// 检查日志目录是否存在
	if err := ensureLogDirectory(logDir); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
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
