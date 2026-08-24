package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

const maxTokenLifetime = 24 * time.Hour

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
	expectedSecret := sha256.Sum256([]byte(l.svcCtx.Config.Auth.AccessSecret))
	providedSecret := sha256.Sum256([]byte(req.SecretKey))
	if subtle.ConstantTimeCompare(expectedSecret[:], providedSecret[:]) != 1 {
		resp.Code = 401
		resp.Msg = "无效的secretKey"
		resp.Data = JwtResponse{Jwt: ""}
		return resp, errors.New("无效的secretKey")
	}
	lifetime := time.Duration(l.svcCtx.Config.Auth.AccessExpire) * time.Second
	if lifetime <= 0 || lifetime > maxTokenLifetime {
		lifetime = maxTokenLifetime
	}
	jwtToken, err := l.getJwtToken(l.svcCtx.Config.Auth.AccessSecret,
		time.Now().Unix(),
		int64(lifetime/time.Second),
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
	claims["nbf"] = iat
	claims["iss"] = "docker-copilot"
	claims["sub"] = "docker-administrator"
	claims["jti"] = uuid.NewString()
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = claims
	return token.SignedString([]byte(secretKey))
}
