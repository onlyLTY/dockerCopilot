package utiles

import (
	"context"
	"errors"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

const dockerAPITimeout = 30 * time.Second

func dockerContext(serviceContext *svc.ServiceContext) (context.Context, context.CancelFunc, error) {
	if serviceContext == nil || serviceContext.DockerClient == nil {
		return nil, nil, errors.New("docker 客户端不可用")
	}
	ctx, cancel := context.WithTimeout(context.Background(), dockerAPITimeout)
	return ctx, cancel, nil
}
