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

	composeTypes "github.com/compose-spec/compose-go/types"
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
	root, content, version, project, err := l.loadCompose(req.ProjectID, req.Filename)
	if err != nil {
		return errorResp(resp, 400, err.Error()), nil
	}
	risks := composeProject.InspectRisks(project, root)
	if err := composeProject.ValidateRisks(project, root, true); err != nil {
		return errorResp(resp, 400, err.Error()), nil
	}
	token := newToken()
	l.svcCtx.ComposeMu.Lock()
	l.svcCtx.ComposeTokens[token] = svc.ComposeToken{ProjectID: req.ProjectID, Filename: req.Filename, Version: version, ExpiresAt: time.Now().Add(10 * time.Minute)}
	l.svcCtx.ComposeMu.Unlock()
	return successResp(resp, map[string]interface{}{
		"projectId": req.ProjectID, "filename": req.Filename, "version": version,
		"contentSize": len(content), "services": project.ServiceNames(), "risks": risks,
		"confirmToken": token,
	}), nil
}

func (l *ActionsLogic) Deploy(req *types.ComposeDeployReq) (*types.Resp, error) {
	resp := &types.Resp{}
	token, ok := l.takeToken(req.ConfirmToken, req.ProjectID, req.Filename)
	if !ok {
		return errorResp(resp, 400, "部署确认已失效，请重新生成预览"), nil
	}
	root, content, version, project, err := l.loadCompose(req.ProjectID, req.Filename)
	if err != nil {
		return errorResp(resp, 400, err.Error()), nil
	}
	if version != token.Version {
		return errorResp(resp, 409, "Compose 文件已变化，请重新生成预览"), nil
	}
	if err := composeProject.ValidateRisks(project, root, req.ConfirmWarnings || l.svcCtx.Config.Compose.AllowHighRisk); err != nil {
		return errorResp(resp, 400, err.Error()), nil
	}
	if !composeRunner.Available(l.ctx) {
		return errorResp(resp, 503, "Docker Compose 插件不可用"), nil
	}
	filePath, err := composeProject.ProjectFilePath(root, req.Filename)
	if err != nil {
		return errorResp(resp, 400, err.Error()), nil
	}
	_ = content
	files := []string{filePath}
	timeout := time.Duration(l.svcCtx.Config.Compose.CommandTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	// 前置校验已通过，转为异步任务执行 compose up，前端通过 taskID 轮询进度。
	taskID := uuid.New().String()
	name := "部署 " + req.ProjectID
	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
		TaskID: taskID, Name: name, Percentage: 0, Message: "任务已提交", DetailMsg: "", IsDone: false,
	})
	svcCtx := l.svcCtx
	go func() {
		defer func() {
			if r := recover(); r != nil {
				svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 0, Message: "部署异常", DetailMsg: fmt.Sprintf("%v", r), IsDone: true})
			}
		}()
		// 独立上下文：请求返回后原 ctx 会被取消，会中断 docker 命令
		bg := context.Background()
		svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 20, Message: "正在检查配置", DetailMsg: "", IsDone: false})
		configResult, err := composeRunner.Config(bg, root, files, timeout)
		if err != nil {
			logx.Errorf("compose config: %v output=%s", err, configResult.Output)
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 20, Message: "配置检查失败", DetailMsg: composeErrMsg("Compose 配置检查失败", err, configResult.Output), IsDone: true})
			return
		}
		svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 50, Message: "正在部署（compose up）", DetailMsg: "", IsDone: false})
		result, err := composeRunner.Up(bg, root, files, timeout)
		if err != nil {
			logx.Errorf("compose up: %v output=%s", err, result.Output)
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 50, Message: "部署失败", DetailMsg: composeErrMsg("Compose 部署失败", err, result.Output), IsDone: true})
			return
		}
		svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 100, Message: "部署完成", DetailMsg: result.Output, IsDone: true})
	}()

	return successResp(resp, map[string]interface{}{"projectId": req.ProjectID, "filename": req.Filename, "taskID": taskID}), nil
}

func (l *ActionsLogic) CleanupPreview(req *types.ComposeCleanupPreviewReq) (*types.Resp, error) {
	resp := &types.Resp{}
	root, err := composeProject.FindProjectRoot(l.svcCtx, req.ProjectID)
	if err != nil {
		return errorResp(resp, 404, err.Error()), nil
	}
	projects, err := composeProject.ScanProjects(l.ctx, l.svcCtx)
	if err != nil {
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
		return errorResp(resp, 409, "项目正在使用或无法判断，不能清理"), nil
	}
	files, err := composeProject.ListProjectFiles(root)
	if err != nil {
		return errorResp(resp, 500, "读取项目文件失败"), nil
	}
	token := newToken()
	l.svcCtx.ComposeMu.Lock()
	l.svcCtx.ComposeTokens[token] = svc.ComposeToken{ProjectID: req.ProjectID, ExpiresAt: time.Now().Add(10 * time.Minute)}
	l.svcCtx.ComposeMu.Unlock()
	backup, err := composeProject.BackupProject(l.svcCtx, root)
	if err != nil {
		return errorResp(resp, 500, "备份项目失败，不能清理"), nil
	}
	return successResp(resp, map[string]interface{}{"projectId": req.ProjectID, "status": target.Status, "files": files, "backup": backup, "previewToken": token}), nil
}

func (l *ActionsLogic) Cleanup(req *types.ComposeCleanupReq) (*types.Resp, error) {
	resp := &types.Resp{}
	if !req.Confirm {
		return errorResp(resp, 400, "必须明确确认清理操作"), nil
	}
	if !l.takeCleanupToken(req.PreviewToken, req.ProjectID) {
		return errorResp(resp, 400, "清理预览已失效，请重新生成"), nil
	}
	root, err := composeProject.FindProjectRoot(l.svcCtx, req.ProjectID)
	if err != nil {
		return errorResp(resp, 404, err.Error()), nil
	}
	projects, err := composeProject.ScanProjects(l.ctx, l.svcCtx)
	if err != nil {
		return errorResp(resp, 500, "扫描 Compose 项目失败"), nil
	}
	for _, project := range projects.Projects {
		if project.ID == req.ProjectID && project.Status != "unused" {
			return errorResp(resp, 409, "项目状态已变化，不能清理"), nil
		}
	}
	if req.DeleteDir {
		entries, err := os.ReadDir(root)
		if err != nil {
			return errorResp(resp, 500, "读取项目目录失败"), nil
		}
		for _, entry := range entries {
			if entry.IsDir() || (entry.Name() != ".env" && !isComposeFile(entry.Name())) {
				return errorResp(resp, 409, "项目目录包含非 Compose 文件，不能删除整个目录"), nil
			}
			info, err := os.Lstat(filepath.Join(root, entry.Name()))
			if err != nil || info.Mode()&os.ModeSymlink != 0 {
				return errorResp(resp, 409, "项目目录包含符号链接，不能删除"), nil
			}
		}
		if err := os.RemoveAll(root); err != nil {
			return errorResp(resp, 500, "删除项目目录失败"), nil
		}
	} else {
		entries, err := os.ReadDir(root)
		if err != nil {
			return errorResp(resp, 500, "读取项目目录失败"), nil
		}
		for _, entry := range entries {
			if entry.IsDir() || (!isComposeFile(entry.Name()) && entry.Name() != ".env") {
				continue
			}
			if err := os.Remove(filepath.Join(root, entry.Name())); err != nil {
				return errorResp(resp, 500, "删除 Compose 文件失败"), nil
			}
		}
	}
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
	detail := strings.TrimSpace(output)
	if detail == "" && err != nil {
		detail = strings.TrimSpace(err.Error())
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
	lower := strings.ToLower(name)
	return lower == "compose.yaml" || lower == "compose.yml" || lower == "docker-compose.yaml" || lower == "docker-compose.yml" || strings.Contains(lower, "override")
}
