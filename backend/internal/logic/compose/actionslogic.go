package compose

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	composeProject "github.com/onlyLTY/dockerCopilot/internal/utiles/compose_project"
	composeRunner "github.com/onlyLTY/dockerCopilot/internal/utiles/compose_runner"
	"github.com/zeromicro/go-zero/core/logx"
)

type ActionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewActionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ActionsLogic {
	return &ActionsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ActionsLogic) DeployPreview(req *types.ComposeDeployPreviewReq) (*types.Resp, error) {
	resp := &types.Resp{}
	logx.Infof("compose operation=deploy_preview project=%s filename=%s", req.ProjectID, req.Filename)
	root, content, version, project, err := l.loadCompose(req.ProjectID, req.Filename)
	if err != nil {
		logx.Errorf("compose operation=deploy_preview project=%s filename=%s failed=load error=%v", req.ProjectID, req.Filename, err)
		return errorResp(resp, 400, clientMsg(err, "读取 Compose 配置失败")), nil
	}
	risks := composeProject.InspectRisks(project, root)
	if err := composeProject.ValidateRisks(project, root, true); err != nil {
		logx.Errorf("compose operation=deploy_preview project=%s filename=%s failed=risk error=%v risks=%d", req.ProjectID, req.Filename, err, len(risks))
		return errorResp(resp, 400, clientMsg(err, "读取 Compose 配置失败")), nil
	}
	token := newToken()
	l.svcCtx.ComposeMu.Lock()
	l.svcCtx.ComposeTokens[token] = svc.ComposeToken{ProjectID: req.ProjectID, Filename: req.Filename, Version: version, ExpiresAt: time.Now().Add(10 * time.Minute)}
	l.svcCtx.ComposeMu.Unlock()
	logx.Infof("compose operation=deploy_preview project=%s filename=%s success services=%d risks=%d bytes=%d", req.ProjectID, req.Filename, len(project.ServiceNames()), len(risks), len(content))
	return successResp(resp, map[string]interface{}{
		"projectId": req.ProjectID, "filename": req.Filename, "version": version,
		"contentSize": len(content), "services": project.ServiceNames(), "risks": risks,
		"confirmToken": token,
	}), nil
}

func (l *ActionsLogic) Deploy(req *types.ComposeDeployReq) (*types.Resp, error) {
	resp := &types.Resp{}
	logx.Infof("compose operation=deploy project=%s filename=%s stage=submit", req.ProjectID, req.Filename)

	// 先占项目锁，再消费 confirm token，避免忙时把一次性 token 白白作废。
	const deployOp = "deploy"
	if !l.svcCtx.TryStartComposeOp(req.ProjectID, deployOp) {
		logx.Errorf("compose operation=deploy project=%s filename=%s failed=project_busy", req.ProjectID, req.Filename)
		return errorResp(resp, 409, "该项目正在部署或清理中，请稍后再试"), nil
	}
	started := false
	defer func() {
		if !started {
			l.svcCtx.FinishComposeOp(req.ProjectID, deployOp)
		}
	}()

	token, ok := l.takeToken(req.ConfirmToken, req.ProjectID, req.Filename)
	if !ok {
		logx.Errorf("compose operation=deploy project=%s filename=%s failed=invalid confirmation", req.ProjectID, req.Filename)
		return errorResp(resp, 400, "部署确认已失效，请重新生成预览"), nil
	}
	root, content, version, project, err := l.loadCompose(req.ProjectID, req.Filename)
	if err != nil {
		logx.Errorf("compose operation=deploy project=%s filename=%s failed=load error=%v", req.ProjectID, req.Filename, err)
		return errorResp(resp, 400, clientMsg(err, "读取 Compose 配置失败")), nil
	}
	if version != token.Version {
		logx.Errorf("compose operation=deploy project=%s filename=%s failed=version conflict", req.ProjectID, req.Filename)
		return errorResp(resp, 409, "Compose 文件已变化，请重新生成预览"), nil
	}
	// 极高危（docker.sock / privileged / 敏感路径）仅 AllowHighRisk 可放行；其余 high 仍可由 confirmWarnings 确认。
	if err := composeProject.ValidateCriticalRisks(project, root, l.svcCtx.Config.Compose.AllowHighRisk); err != nil {
		logx.Errorf("compose operation=deploy project=%s filename=%s failed=critical risk error=%v", req.ProjectID, req.Filename, err)
		return errorResp(resp, 400, clientMsg(err, "存在极高危配置，无法部署")), nil
	}
	if err := composeProject.ValidateRisks(project, root, req.ConfirmWarnings || l.svcCtx.Config.Compose.AllowHighRisk); err != nil {
		logx.Errorf("compose operation=deploy project=%s filename=%s failed=risk error=%v", req.ProjectID, req.Filename, err)
		return errorResp(resp, 400, clientMsg(err, "存在高风险配置，请确认后部署")), nil
	}
	if !composeRunner.Available(l.ctx, l.svcCtx.DockerClient) {
		logx.Errorf("compose operation=deploy project=%s filename=%s failed=docker daemon unavailable", req.ProjectID, req.Filename)
		return errorResp(resp, 503, "Docker 服务不可用"), nil
	}
	filePath, err := composeProject.ProjectFilePath(root, req.Filename)
	if err != nil {
		logx.Errorf("compose operation=deploy project=%s filename=%s failed=path error=%v", req.ProjectID, req.Filename, err)
		return errorResp(resp, 400, clientMsg(err, "Compose 文件路径无效")), nil
	}
	_ = content
	files := []string{filePath}
	deploymentMappings := make([]composeRunner.PathMapping, 0)
	for _, mapping := range composeProject.DeploymentPathMappings(l.ctx, l.svcCtx) {
		deploymentMappings = append(deploymentMappings, composeRunner.PathMapping{
			HostPath: mapping.HostPath, ContainerPath: mapping.ContainerPath,
		})
	}
	timeout := time.Duration(l.svcCtx.Config.Compose.CommandTimeoutSec) * time.Second

	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	taskID := uuid.New().String()
	name := "部署 " + req.ProjectID
	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
		TaskID: taskID, Name: name, Percentage: 0, Message: "任务已提交", DetailMsg: "", IsDone: false,
	})
	logx.Infof("compose operation=deploy project=%s filename=%s task=%s stage=submit success", req.ProjectID, req.Filename, taskID)
	svcCtx := l.svcCtx
	projectID := req.ProjectID
	filename := req.Filename
	pullImages := req.PullImages
	started = true // 锁交给异步任务释放
	go func() {
		defer svcCtx.FinishComposeOp(projectID, deployOp)
		defer func() {
			if r := recover(); r != nil {
				logx.Errorf("compose operation=deploy project=%s filename=%s task=%s stage=panic failed=%v", projectID, filename, taskID, r)
				logx.Errorf("compose deploy panic detail=%v", r)
				svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 0, Message: "部署异常", DetailMsg: "部署过程发生内部错误，请查看服务日志", IsDone: true})
			}
		}()
		bg, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 20, Message: "正在检查配置", DetailMsg: "", IsDone: false})

		configResult, err := composeRunner.Config(bg, svcCtx.DockerClient, root, files, timeout)
		if err != nil {
			logx.Errorf("compose operation=deploy project=%s filename=%s task=%s stage=config failed=%v output=%q", projectID, filename, taskID, err, configResult.Output)
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 20, Message: "配置检查失败", DetailMsg: composeErrMsg("Compose 配置检查失败", err, configResult.Output), IsDone: true})
			return
		}
		logx.Infof("compose operation=deploy project=%s filename=%s task=%s stage=config success", projectID, filename, taskID)
		svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 50, Message: "正在部署（compose up）", DetailMsg: "", IsDone: false})
		result, err := composeRunner.UpWithProgress(bg, svcCtx.DockerClient, root, files, timeout, pullImages, func(event composeRunner.ProgressEvent) {
			message := event.Message
			percentage := 50
			if event.IsPull {
				message = "正在拉取新镜像"
				percentage = 0
			}
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{
				TaskID: taskID, Name: name, Percentage: percentage, Message: message, DetailMsg: event.Message, StepPercentage: event.StepPercentage, IsDone: false,
			})
		}, deploymentMappings...)

		if err != nil {
			logx.Errorf("compose operation=deploy project=%s filename=%s task=%s stage=up failed=%v output=%q", projectID, filename, taskID, err, result.Output)
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 50, Message: "部署失败", DetailMsg: composeErrMsg("Compose 部署失败", err, result.Output), IsDone: true})
			return
		}
		logx.Infof("compose operation=deploy project=%s filename=%s task=%s stage=up success", projectID, filename, taskID)
		svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 100, Message: "部署完成", DetailMsg: "", IsDone: true})
	}()

	return successResp(resp, map[string]interface{}{"projectId": req.ProjectID, "filename": req.Filename, "taskID": taskID}), nil
}

func (l *ActionsLogic) CleanupPreview(req *types.ComposeCleanupPreviewReq) (*types.Resp, error) {
	resp := &types.Resp{}
	logx.Infof("compose operation=cleanup_preview project=%s", req.ProjectID)
	root, err := composeProject.FindProjectRoot(l.svcCtx, req.ProjectID)
	if err != nil {
		logx.Errorf("compose operation=cleanup_preview project=%s failed=find_root error=%v", req.ProjectID, err)
		return errorResp(resp, 404, clientMsg(err, "Compose 项目不存在")), nil
	}
	projects, err := composeProject.ScanProjects(l.ctx, l.svcCtx)
	if err != nil {
		logx.Errorf("compose operation=cleanup_preview project=%s failed=scan error=%v", req.ProjectID, err)
		return errorResp(resp, 500, "扫描 Compose 项目失败"), nil
	}
	var target *types.ComposeProject
	for i := range projects.Projects {
		if projects.Projects[i].ID == req.ProjectID {
			target = &projects.Projects[i]
			break
		}
	}
	if target == nil || target.Status != "unused" {
		logx.Errorf("compose operation=cleanup_preview project=%s failed=project_not_unused", req.ProjectID)
		return errorResp(resp, 409, "项目正在使用或无法判断，不能清理"), nil
	}
	files, err := composeProject.ListProjectFiles(root)
	if err != nil {
		logx.Errorf("compose operation=cleanup_preview project=%s failed=list_files error=%v", req.ProjectID, err)
		return errorResp(resp, 500, "读取项目文件失败"), nil
	}
	token := newToken()
	l.svcCtx.ComposeMu.Lock()
	l.svcCtx.ComposeTokens[token] = svc.ComposeToken{ProjectID: req.ProjectID, ExpiresAt: time.Now().Add(10 * time.Minute)}
	l.svcCtx.ComposeMu.Unlock()
	backup, err := composeProject.BackupProject(l.svcCtx, root)
	if err != nil {
		logx.Errorf("compose operation=cleanup_preview project=%s failed=backup error=%v", req.ProjectID, err)
		return errorResp(resp, 500, "备份项目失败，不能清理"), nil
	}
	logx.Infof("compose operation=cleanup_preview project=%s success files=%d", req.ProjectID, len(files))
	return successResp(resp, map[string]interface{}{"projectId": req.ProjectID, "status": target.Status, "files": files, "backup": backup, "previewToken": token}), nil
}

func (l *ActionsLogic) Cleanup(req *types.ComposeCleanupReq) (*types.Resp, error) {
	resp := &types.Resp{}
	logx.Infof("compose operation=cleanup project=%s delete_dir=%t stage=submit", req.ProjectID, req.DeleteDir)
	if !req.Confirm {
		logx.Errorf("compose operation=cleanup project=%s failed=confirmation_required", req.ProjectID)
		return errorResp(resp, 400, "必须明确确认清理操作"), nil
	}
	// 先占项目锁再消费 preview token，避免忙时把一次性 token 作废。
	const cleanupOp = "cleanup"
	if !l.svcCtx.TryStartComposeOp(req.ProjectID, cleanupOp) {
		logx.Errorf("compose operation=cleanup project=%s failed=project_busy", req.ProjectID)
		return errorResp(resp, 409, "该项目正在部署或清理中，请稍后再试"), nil
	}
	defer l.svcCtx.FinishComposeOp(req.ProjectID, cleanupOp)

	if !l.takeCleanupToken(req.PreviewToken, req.ProjectID) {
		logx.Errorf("compose operation=cleanup project=%s failed=invalid_preview", req.ProjectID)
		return errorResp(resp, 400, "清理预览已失效，请重新生成"), nil
	}

	root, err := composeProject.FindProjectRoot(l.svcCtx, req.ProjectID)
	if err != nil {
		logx.Errorf("compose operation=cleanup project=%s failed=find_root error=%v", req.ProjectID, err)
		return errorResp(resp, 404, clientMsg(err, "Compose 项目不存在")), nil
	}
	projects, err := composeProject.ScanProjects(l.ctx, l.svcCtx)
	if err != nil {
		logx.Errorf("compose operation=cleanup project=%s failed=scan error=%v", req.ProjectID, err)
		return errorResp(resp, 500, "扫描 Compose 项目失败"), nil
	}
	targetFound := false
	for _, project := range projects.Projects {
		if project.ID != req.ProjectID {
			continue
		}
		targetFound = true
		if project.Status != "unused" {
			logx.Errorf("compose operation=cleanup project=%s failed=project_status_changed status=%s", req.ProjectID, project.Status)
			return errorResp(resp, 409, "项目状态已变化，不能清理"), nil
		}
		break
	}
	if !targetFound {
		logx.Errorf("compose operation=cleanup project=%s failed=project_not_found_after_scan", req.ProjectID)
		return errorResp(resp, 409, "项目状态已变化或项目不存在，不能清理"), nil
	}
	if req.DeleteDir {
		if err := composeProject.RemoveProjectRoot(root, l.svcCtx.Config.Compose.ScanPaths); err != nil {
			logx.Errorf("compose operation=cleanup project=%s stage=remove_dir_validate failed=%v", req.ProjectID, err)
			return errorResp(resp, 409, "项目目录不安全，不能删除整个目录"), nil
		}
		if _, err := os.Lstat(root); err == nil {
			logx.Errorf("compose operation=cleanup project=%s stage=remove_dir failed=directory_still_exists", req.ProjectID)
			return errorResp(resp, 500, "删除项目目录失败"), nil
		} else if !os.IsNotExist(err) {
			logx.Errorf("compose operation=cleanup project=%s stage=remove_dir failed=%v", req.ProjectID, err)
			return errorResp(resp, 500, "删除项目目录失败"), nil
		}
	} else {
		entries, err := os.ReadDir(root)
		if err != nil {
			logx.Errorf("compose operation=cleanup project=%s stage=read_dir failed=%v", req.ProjectID, err)
			return errorResp(resp, 500, "读取项目目录失败"), nil
		}
		for _, entry := range entries {
			if entry.IsDir() || (!isComposeFile(entry.Name()) && !composeProject.IsComposeEnvFile(entry.Name())) {
				continue
			}
			if err := os.Remove(filepath.Join(root, entry.Name())); err != nil {
				logx.Errorf("compose operation=cleanup project=%s stage=remove_file file=%s failed=%v", req.ProjectID, entry.Name(), err)
				return errorResp(resp, 500, "删除 Compose 文件失败"), nil
			}
		}
	}
	logx.Infof("compose operation=cleanup project=%s success delete_dir=%t", req.ProjectID, req.DeleteDir)
	return successResp(resp, map[string]interface{}{"projectId": req.ProjectID, "deleted": true}), nil
}

func (l *ActionsLogic) loadCompose(id, filename string) (string, []byte, string, *composeTypes.Project, error) {
	root, err := composeProject.FindProjectRoot(l.svcCtx, id)
	if err != nil {
		return "", nil, "", nil, err
	}
	content, version, err := composeProject.ReadProjectFile(root, filename)
	if err != nil {
		return "", nil, "", nil, err
	}
	project, err := composeProject.ParseComposeContent(root, filename, content)
	if err != nil {
		return "", nil, "", nil, err
	}
	return root, content, version, project, nil
}

func (l *ActionsLogic) takeToken(token, projectID, filename string) (svc.ComposeToken, bool) {
	l.svcCtx.ComposeMu.Lock()
	defer l.svcCtx.ComposeMu.Unlock()
	value, ok := l.svcCtx.ComposeTokens[token]
	if !ok || value.ExpiresAt.Before(time.Now()) || value.ProjectID != projectID || value.Filename != filename {
		return svc.ComposeToken{}, false
	}
	delete(l.svcCtx.ComposeTokens, token)
	return value, true
}

func (l *ActionsLogic) takeCleanupToken(token, projectID string) bool {
	l.svcCtx.ComposeMu.Lock()
	defer l.svcCtx.ComposeMu.Unlock()
	value, ok := l.svcCtx.ComposeTokens[token]
	if !ok || value.ExpiresAt.Before(time.Now()) || value.ProjectID != projectID {
		return false
	}
	delete(l.svcCtx.ComposeTokens, token)
	return true
}

func newToken() string {
	data := make([]byte, 24)
	if _, err := rand.Read(data); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(data)
}

// composeErrMsg 组合用户可读的前缀与 docker 实际输出，便于前端直接展示真实原因。
// output 已在 compose_runner 中做过敏感信息脱敏与长度限制。
func composeErrMsg(prefix string, err error, output string) string {
	// output 已经过 compose_runner 脱敏；无输出时仅用 client 安全摘要，不回传原始 err
	detail := strings.TrimSpace(output)
	if detail == "" && err != nil {
		detail = clientMsg(err, "")
		if detail == "操作失败" {
			detail = ""
		}
	}
	if detail == "" {
		return prefix
	}
	const maxDetail = 2000
	if len(detail) > maxDetail {
		detail = detail[:maxDetail] + "…"
	}
	return prefix + "：" + detail
}

func isComposeFile(name string) bool {
	return composeProject.IsComposeFile(name)
}
