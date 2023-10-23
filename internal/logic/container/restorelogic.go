package container

import (
	"context"
	"github.com/google/uuid"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/utiles"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RestoreLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRestoreLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RestoreLogic {
	return &RestoreLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RestoreLogic) Restore(req *types.ContainerRestoreReq) (resp *types.Resp, err error) {
	// todo: add your logic here and delete this line
	resp = &types.Resp{}
	taskID := uuid.New().String()
	go func() {
		// Catch any panic and log the error
		defer func() {
			if r := recover(); r != nil {
				logx.Errorf("Recovered from panic in restoreContainer: %v", r)
			}
		}()
		err := utiles.RestoreContainer(l.svcCtx, req.Filename, taskID)
		if err != nil {
			logx.Errorf("Error in restoreContainer: %v", err)
		}
	}()
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = taskID
	return resp, nil
}
