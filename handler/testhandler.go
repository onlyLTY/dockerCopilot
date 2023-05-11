package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/onlyLTY/oneKeyUpdate/v2/logic"
	"github.com/onlyLTY/oneKeyUpdate/v2/svc"
)

func TestHandler(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(context *gin.Context) {
		l := logic.NewTestLogic(context, svcCtx)
		resp, err := l.Getnotice()
		if err != nil {
			context.AsciiJSON(500, "server error")
		} else {
			context.AsciiJSON(200, resp)
		}
	}
}
