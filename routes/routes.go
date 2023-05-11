package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/onlyLTY/oneKeyUpdate/v2/handler"
	"github.com/onlyLTY/oneKeyUpdate/v2/svc"
)

func RegisterHandlers(r *gin.Engine, svcCtx *svc.ServiceContext) {
	testRouters := r.Group("/test")
	{
		testRouters.GET("/", handler.TestHandler(svcCtx))
	}

}
