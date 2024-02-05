package main

import (
	"embed"
	"flag"
	"fmt"
	loader "github.com/nathan-osman/pongo2-embed-loader"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/handler"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/utiles"
	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/x/errors"
	xhttp "github.com/zeromicro/x/http"
	"go/types"
	"net/http"
	"os"
	"strings"

	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/config"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/dockerCopilot.yaml", "the config file")

type UnauthorizedResponse struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data map[string]interface{} `json:"data"`
}

//go:embed templates/*
var content embed.FS

func main() {
	flag.Parse()

	var c config.Config
	err := conf.Load(*configFile, &c, conf.UseEnv())
	if err != nil {
		logx.Errorf("无法加载配置文件出错: %v", err)
		logx.Errorf("请确认secretKey设置正确，要求非纯数字且大于八位")
		os.Exit(1)
	}
	server := rest.MustNewServer(c.RestConf, rest.WithCors("*"), rest.WithUnauthorizedCallback(
		func(w http.ResponseWriter, r *http.Request, err error) {
			response := UnauthorizedResponse{
				Code: http.StatusUnauthorized, // 401
				Msg:  "未授权",
				Data: map[string]interface{}{},
			}
			httpx.WriteJson(w, http.StatusUnauthorized, response)
		}))
	defer server.Stop()
	ctx := svc.NewServiceContext(c, &loader.Loader{Content: content})
	list, err := utiles.GetImagesList(ctx)
	if err != nil {
		logx.Errorf("panic获取镜像列表出错: %v", err)
		panic(err)
	}
	go ctx.HubImageInfo.CheckUpdate(list)
	corndanmu := cron.New(cron.WithParser(cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
	)))
	_, err = corndanmu.AddFunc("30 * * * *", func() {
		list, err := utiles.GetImagesList(ctx)
		if err != nil {
			logx.Errorf("panic获取镜像列表出错: %v", err)
			panic(err)
		}
		ctx.HubImageInfo.CheckUpdate(list)
	})
	if err != nil {
		logx.Errorf("panic添加定时任务出错: %v", err)
		panic(err)
	}
	corndanmu.Start()
	defer corndanmu.Stop()
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

	//这里注册
	dirlevel := []string{":1", ":2", ":3", ":4", ":5", ":6", ":7", ":8"}
	patern := "/static/"
	dirpath := "./static/"
	for i := 1; i < len(dirlevel); i++ {
		path := patern + strings.Join(dirlevel[:i], "/")
		//最后生成 /asset
		engine.AddRoute(
			rest.Route{
				Method:  http.MethodGet,
				Path:    path,
				Handler: http.StripPrefix(patern, http.FileServer(http.Dir(dirpath))).ServeHTTP,
			})

		//logx.Infof("register dir  %s  %s", path, dirpath)
	}

	engine.AddRoute(
		rest.Route{
			Method: http.MethodGet,
			Path:   "/manager/",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "./templates/index.html")
			},
		},
	)

	managerPatern := "/manager/"
	managerDirpath := "./templates/"
	for i := 1; i < len(dirlevel); i++ {
		path := managerPatern + strings.Join(dirlevel[:i], "/")
		//最后生成 /asset
		engine.AddRoute(
			rest.Route{
				Method:  http.MethodGet,
				Path:    path,
				Handler: http.StripPrefix(managerPatern, http.FileServer(http.Dir(managerDirpath))).ServeHTTP,
			})

		//logx.Infof("register dir  %s  %s", path, dirpath)
	}
}
