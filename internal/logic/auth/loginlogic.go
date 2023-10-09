package auth

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	if l.svcCtx.Config.SecretKey != req.SecretKey {
		return nil, errors.New("密钥错误")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * 720).Unix(),
	})
	tokenString, err := token.SignedString([]byte(req.SecretKey))
	if err != nil {
		return nil, errors.New("无法生成 token，请重试")
	}
	return &types.LoginResp{Token: tokenString}, nil
}
