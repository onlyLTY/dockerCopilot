package auth

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"time"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type JwtResponse struct {
	Jwt string `json:"jwt"`
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	if l.svcCtx.Config.SecretKey != req.SecretKey {
		resp.Code = 401
		resp.Msg = "无效的secretKey"
		return resp, errors.New("无效的secretKey")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * 720).Unix(),
	})
	jwtString, err := token.SignedString([]byte(req.SecretKey))
	if err != nil {
		resp.Code = 500
		resp.Msg = "无法生成 token，请重试"
		return resp, errors.New("生成 token出现错误，请重试")
	}
	resp.Code = 201
	resp.Msg = "success"
	resp.Data = JwtResponse{Jwt: jwtString}
	return resp, nil
}
