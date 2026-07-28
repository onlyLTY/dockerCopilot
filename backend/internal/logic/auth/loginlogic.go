package auth

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
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
	if l.svcCtx.Config.Auth.AccessSecret != req.SecretKey {
		resp.Code = 401
		resp.Msg = "无效的secretKey"
		resp.Data = JwtResponse{Jwt: ""}
		return resp, errors.New("无效的secretKey")
	}
	jwtToken, err := l.getJwtToken(l.svcCtx.Config.Auth.AccessSecret,
		time.Now().Unix(),
		l.svcCtx.Config.Auth.AccessExpire,
	)
	if err != nil {
		resp.Code = 500
		resp.Msg = "无法生成 token，请重试"
		resp.Data = JwtResponse{Jwt: ""}
		return resp, errors.New("生成 token出现错误，请重试")
	}
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = JwtResponse{Jwt: jwtToken}
	return resp, nil
}

func (l *LoginLogic) getJwtToken(secretKey string, iat, seconds int64) (string, error) {
	claims := make(jwt.MapClaims)
	claims["iat"] = iat
	claims["exp"] = iat + seconds
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = claims
	return token.SignedString([]byte(secretKey))
}
