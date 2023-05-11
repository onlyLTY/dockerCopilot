package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/onlyLTY/oneKeyUpdate/v2/config"
	"github.com/onlyLTY/oneKeyUpdate/v2/routes"
	"github.com/onlyLTY/oneKeyUpdate/v2/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var configFile = flag.String("f", "etc/onekeyupdate.yaml", "the config file")

func main() {
	flag.Parse()
	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	logx.MustSetup(c.Log)
	logx.DisableStat()
	ctx := svc.NewServiceContext(c)
	r := gin.Default()
	routes.RegisterHandlers(r, ctx)
	err := r.Run(fmt.Sprintf("%s:%d", c.Host, c.Port))
	if err != nil {
		logx.Error(err)
	}
}
