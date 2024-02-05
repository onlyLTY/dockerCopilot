package utiles

import (
	"context"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

func RenameContainer(ctx *svc.ServiceContext, id string, newName string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logx.Errorf("connect to docker error: %v", err)
		return err
	}
	err = cli.ContainerRename(context.TODO(), id, newName)
	if err != nil {
		return err
	}
	return nil
}
