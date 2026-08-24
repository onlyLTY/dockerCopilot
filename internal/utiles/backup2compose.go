package utiles

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	backupCompose "github.com/onlyLTY/dockerCopilot/internal/utiles/backup_compose"
	"github.com/zeromicro/go-zero/core/logx"
)

func Backup2Compose(ctx *svc.ServiceContext) (err error) {
	containerList, err := GetContainerList(ctx)
	if err != nil {
		return err
	}
	containerJSONs := make([]container.InspectResponse, 0, len(containerList))
	for _, v := range containerList {
		containerID := v.ID
		inspectedContainer, err := GetContainerInspect(ctx, containerID)
		if err != nil {
			logx.Error("获取容器信息失败" + err.Error())
			return err
		}
		containerJSONs = append(containerJSONs, inspectedContainer)
	}
	includeSensitiveEnvironment := false
	if raw := strings.TrimSpace(os.Getenv("COMPOSE_BACKUP_INCLUDE_SECRETS")); raw != "" {
		parsed, parseErr := strconv.ParseBool(raw)
		if parseErr != nil {
			return fmt.Errorf("COMPOSE_BACKUP_INCLUDE_SECRETS 配置错误: %w", parseErr)
		}
		includeSensitiveEnvironment = parsed
	}
	composeYAML, err := backupCompose.DockerConfig2ComposeYamlWithOptions(containerJSONs, backupCompose.Options{
		IncludeSensitiveEnvironment: includeSensitiveEnvironment,
	})
	if err != nil {
		logx.Error("备份失败" + err.Error())
		return err
	}
	backupDir, err := ensureBackupDir()
	if err != nil {
		return err
	}
	if err := writeBackupAtomic(backupDir, newBackupFilename(".yaml"), composeYAML); err != nil {
		logx.Errorf("写入 Compose 备份失败: %v", err)
		return err
	}
	return nil
}
