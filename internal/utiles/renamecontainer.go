package utiles

import (
	"context"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
)

func RenameContainer(ctx *svc.ServiceContext, oldName string, newName string) (types.MsgResp, error) {
	containers, err := GetContainerList(ctx)
	if err != nil {
		return types.MsgResp{}, err
	}
	containerID, err := findContainerIDByName(containers, oldName)
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	err = cli.ContainerRename(context.TODO(), containerID, newName)
	if err != nil {
		return types.MsgResp{Msg: err.Error()}, err
	}
	return types.MsgResp{}, nil
}
