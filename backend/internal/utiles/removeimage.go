package utiles

import (
	"context"

	"github.com/docker/docker/api/types/image"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func RemoveImage(ctx *svc.ServiceContext, target string, force bool) error {
	return RemoveImageWithContext(context.Background(), ctx, target, force)
}

func RemoveImageWithContext(taskCtx context.Context, ctx *svc.ServiceContext, target string, force bool) error {
	if err := requireDocker(ctx); err != nil {
		return err
	}
	if !ctx.AcquireImageOp(taskCtx) {
		if err := taskCtx.Err(); err != nil {
			return err
		}
		return context.Canceled
	}
	defer ctx.ReleaseImageOp()
	_, err := ctx.DockerClient.ImageRemove(taskCtx, target, image.RemoveOptions{Force: force})
	return err
}
