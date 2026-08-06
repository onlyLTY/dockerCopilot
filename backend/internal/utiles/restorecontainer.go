package utiles

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	dockerBackend "github.com/docker/docker/api/types/backend"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

func RestoreContainer(ctx *svc.ServiceContext, filename string, taskID string) error {
	if err := requireDocker(ctx); err != nil {
		return err
	}
	var backupList []string
	fullPath, err := ResolveBackupPath(filename)
	if err != nil {
		return err
	}
	oldProgress := svc.TaskProgress{
		TaskID:     taskID,
		Percentage: 0,
		Message:    "",
		Name:       "",
		DetailMsg:  "",
		IsDone:     false,
	}
	oldProgress.Name = "恢复容器"
	content, err := os.ReadFile(fullPath)
	if err != nil {
		logx.Errorf("Failed to read file: %s", err)
		oldProgress.Percentage = 0
		oldProgress.Message = "读取文件失败或者未找到文件"
		oldProgress.DetailMsg = "读取备份文件失败"
		oldProgress.IsDone = true
		ctx.UpdateProgress(taskID, oldProgress)
		return err
	}
	var configList []dockerBackend.ContainerCreateConfig
	err = json.Unmarshal(content, &configList)
	if err != nil {
		logx.Errorf("Failed to parse json: %s", err)
		oldProgress.Percentage = 0
		oldProgress.Message = "解析文件失败"
		oldProgress.DetailMsg = "备份内容不是有效的容器配置 JSON"
		oldProgress.IsDone = true
		ctx.UpdateProgress(taskID, oldProgress)
		return err
	}
	total := len(configList)
	for i, containerInfo := range configList {
		name := containerInfo.Name
		if name == "" {
			name = "container-" + strconv.Itoa(i+1)
		}
		linePrefix := fmt.Sprintf("[%d/%d] %s", i+1, total, name)
		info := "正在恢复 " + linePrefix
		oldProgress.Percentage = int(float64(i) / float64(total) * 100)
		oldProgress.Message = info
		// 进行中：已完成行 + 当前行，每容器一行便于阅读
		oldProgress.DetailMsg = joinRestoreLines(backupList, linePrefix+" …")
		ctx.UpdateProgress(taskID, oldProgress)
		ctx.DockerClient.NegotiateAPIVersion(context.TODO())

		// 与 Compose 部署门禁对齐：极高危默认拒绝；高风险（host 网络/PID、devices、cap_add 等）
		// 亦默认拒绝。仅当服务端 AllowHighRisk=true 时放行（对齐部署侧无 confirm 时的硬拦策略）。
		if blockErr := validateRestoreHostConfig(name, containerInfo.HostConfig, ctx.Config.Compose.AllowHighRisk); blockErr != nil {
			logx.Errorf("拒绝高危恢复配置 name=%s: %v", name, blockErr)
			backupList = append(backupList, linePrefix+" 恢复被拒绝（高危配置）")
			continue
		}

		if containerInfo.Config == nil || containerInfo.Config.Image == "" {
			backupList = append(backupList, linePrefix+" 恢复失败：备份缺少镜像信息")
			continue
		}
		reader, err := ctx.DockerClient.ImagePull(context.TODO(), containerInfo.Config.Image, image.PullOptions{})
		if err != nil {
			logx.Errorf("Failed to pull image: %s", err)
			backupList = append(backupList, linePrefix+" 拉取镜像失败")
			continue
		}
		err = decodePullResp(reader, ctx, taskID)
		_ = reader.Close()
		if err != nil {
			logx.Errorf("Failed to pull image: %s", err)
			backupList = append(backupList, linePrefix+" 拉取镜像失败")
			continue
		}
		_, err = ctx.DockerClient.ContainerCreate(context.TODO(), containerInfo.Config, containerInfo.HostConfig, containerInfo.NetworkingConfig, nil, containerInfo.Name)
		if err != nil {
			logx.Errorf("Failed to create container: %s", err)
			backupList = append(backupList, linePrefix+" 恢复失败")
			continue
		}
		backupList = append(backupList, linePrefix+" 恢复成功")
	}
	oldProgress.Percentage = 100
	oldProgress.DetailMsg = joinRestoreLines(backupList)
	oldProgress.Message = "恢复完成"
	oldProgress.IsDone = true
	ctx.UpdateProgress(taskID, oldProgress)
	return nil
}

// joinRestoreLines 将恢复结果按行拼接；extra 为当前进行中的一行（可选）。
func joinRestoreLines(done []string, extra ...string) string {
	parts := make([]string, 0, len(done)+len(extra))
	parts = append(parts, done...)
	parts = append(parts, extra...)
	return strings.Join(parts, "\n")
}

// validateRestoreHostConfig 拦截备份恢复中的高危 HostConfig，规则与 Compose 部署门禁对齐：
//   - 极高危（privileged / docker.sock / 敏感路径）：默认拒绝；AllowHighRisk 才放行
//   - 高风险（host 网络/PID、devices、cap_add、security_opt）：默认拒绝；AllowHighRisk 才放行
//
// 恢复流程没有前端 confirmWarnings，因此「默认拒绝」等价于部署时未勾选确认的行为。
func validateRestoreHostConfig(name string, hc *container.HostConfig, allowHighRisk bool) error {
	if hc == nil {
		return nil
	}
	if allowHighRisk {
		return nil
	}

	// —— 极高危（对应 compose RiskKindPrivileged / DockerSocket / SensitivePath）——
	if hc.Privileged {
		return fmt.Errorf("容器 %s 启用了 privileged，已拒绝恢复", name)
	}
	for _, b := range hc.Binds {
		src := BindSource(b)
		if IsDockerSocketPath(src) {
			return fmt.Errorf("容器 %s 挂载了 Docker socket，已拒绝恢复", name)
		}
		if IsSensitiveHostPath(src) {
			return fmt.Errorf("容器 %s 挂载了敏感宿主机路径，已拒绝恢复", name)
		}
	}
	for _, m := range hc.Mounts {
		src := m.Source
		if IsDockerSocketPath(src) {
			return fmt.Errorf("容器 %s 挂载了 Docker socket，已拒绝恢复", name)
		}
		if IsSensitiveHostPath(src) {
			return fmt.Errorf("容器 %s 挂载了敏感宿主机路径，已拒绝恢复", name)
		}
	}

	// —— 高风险（对应 compose host_network / host_pid / devices / cap_add / security_opt）——
	if string(hc.NetworkMode) == "host" {
		return fmt.Errorf("容器 %s 使用 host 网络，已拒绝恢复", name)
	}
	if string(hc.PidMode) == "host" {
		return fmt.Errorf("容器 %s 共享宿主机 PID 命名空间，已拒绝恢复", name)
	}
	if len(hc.Devices) > 0 {
		return fmt.Errorf("容器 %s 使用了宿主机设备，已拒绝恢复", name)
	}
	if len(hc.CapAdd) > 0 {
		return fmt.Errorf("容器 %s 增加了 Linux capabilities，已拒绝恢复", name)
	}
	if len(hc.SecurityOpt) > 0 {
		return fmt.Errorf("容器 %s 配置了 security_opt，已拒绝恢复", name)
	}
	return nil
}
