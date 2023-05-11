package logic

import (
	"github.com/gin-gonic/gin"
	"github.com/onlyLTY/oneKeyUpdate/v2/svc"
	"github.com/onlyLTY/oneKeyUpdate/v2/types"
)

type TestLogic struct {
	ctx    *gin.Context
	svcCtx *svc.ServiceContext
}

func NewTestLogic(ctx *gin.Context, svcCtx *svc.ServiceContext) *TestLogic {
	return &TestLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TestLogic) Getnotice() (resp *types.Test, err error) {
	return &types.Test{Code: 20000, Message: "hello world"}, nil
}
