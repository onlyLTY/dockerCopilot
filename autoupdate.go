package main

import (
	"embed"
	"flag"
	"fmt"
	loader "github.com/nathan-osman/pongo2-embed-loader"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/utiles"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/x/errors"
	xhttp "github.com/zeromicro/x/http"
	"go/types"
	"net/http"
	"strings"

	"github.com/onlyLTY/oneKeyUpdate/v2/internal/config"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/handler"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/autoupdate.yaml", "the config file")

//go:embed templates/*
var content embed.FS

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()
	ctx := svc.NewServiceContext(c, &loader.Loader{Content: content})
	_, _, err := utiles.GetNewJwt(ctx)
	if err != nil {
		panic(err)
	}
	handler.RegisterHandlers(server, ctx)
	RegisterHandlers(server)
	httpx.SetErrorHandler(func(err error) (int, any) {
		switch e := err.(type) {
		case *errors.CodeMsg:
			return http.StatusOK, xhttp.BaseResponse[types.Nil]{
				Code: e.Code,
				Msg:  e.Msg,
			}
		default:
			return http.StatusInternalServerError, err
		}
	})
	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
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
}
