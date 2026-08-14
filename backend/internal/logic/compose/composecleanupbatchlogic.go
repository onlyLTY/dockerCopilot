package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	composeProject "github.com/onlyLTY/dockerCopilot/internal/utiles/compose_project"
)

const composeCleanupBatchTaskName = "批量删除 Compose 项目"

func NewComposeCleanupBatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComposeCleanupBatchLogic {
	return &ComposeCleanupBatchLogic{ctx: ctx, svcCtx: svcCtx}
}

type ComposeCleanupBatchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func (l *ComposeCleanupBatchLogic) Submit(req *types.ComposeCleanupBatchReq) (*types.Resp, error) {
	resp := &types.Resp{}
	if !req.Confirm {
		return errorResp(resp, 400, "必须明确确认清理操作"), nil
	}
	ids := make([]string, 0, len(req.ProjectIDs))
	seen := make(map[string]struct{}, len(req.ProjectIDs))
	for _, id := range req.ProjectIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return errorResp(resp, 400, "请选择要删除的 Compose 项目"), nil
	}
	taskID := uuid.NewString()
	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Refresh: true, Name: composeCleanupBatchTaskName, Message: "任务已提交"})
	taskCtx := l.svcCtx.RegisterTask(taskID)
	go runComposeCleanupBatch(taskCtx, l.svcCtx, taskID, ids, req.DeleteDir)
	return successResp(resp, map[string]interface{}{"taskID": taskID}), nil
}

func runComposeCleanupBatch(taskCtx context.Context, svcCtx *svc.ServiceContext, taskID string, ids []string, deleteDir bool) {
	defer svcCtx.FinishTask(taskID)
	defer func() {
		if recovered := recover(); recovered != nil {
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: composeCleanupBatchTaskName, Percentage: 100, Message: "批量删除失败", DetailMsg: fmt.Sprintf("任务异常：%v", recovered), Failed: true, IsDone: true})
		}
	}()
	failed := make([]string, 0)
	for index, id := range ids {
		if taskCtx.Err() != nil {
			svcCtx.MarkTaskCanceled(taskID, "用户请求停止批量删除 Compose 项目")
			return
		}
		name := "删除项目 " + id
		if !svcCtx.TryStartComposeOp(id, "cleanup") {
			err := fmt.Errorf("项目正在部署、清理、保存或备份中")
			failed = append(failed, id+"："+err.Error())
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: composeCleanupBatchTaskName, Message: name, DetailMsg: err.Error(), Failed: true})
			continue
		}
		var err error
		func() {
			defer svcCtx.FinishComposeOp(id, "cleanup")
			err = cleanupComposeProject(taskCtx, svcCtx, id, deleteDir)
		}()
		percentage := (index + 1) * 100 / len(ids)
		if err != nil {
			if taskCtx.Err() != nil {
				svcCtx.MarkTaskCanceled(taskID, "用户请求停止批量删除 Compose 项目")
				return
			}
			failed = append(failed, id+"："+err.Error())
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: composeCleanupBatchTaskName, Percentage: percentage, Message: name, DetailMsg: "删除失败：" + err.Error(), Failed: true})
			continue
		}
		svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: composeCleanupBatchTaskName, Percentage: percentage, Message: name, DetailMsg: "删除完成"})
	}
	if taskCtx.Err() != nil {
		svcCtx.MarkTaskCanceled(taskID, "用户请求停止批量删除 Compose 项目")
		return
	}
	detail := "全部 Compose 项目删除完成"
	if len(failed) > 0 {
		detail = "失败项目：" + strings.Join(failed, "；")
	}
	svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: composeCleanupBatchTaskName, Percentage: 100, Message: "批量删除完成", DetailMsg: detail, Failed: len(failed) > 0, IsDone: true})
}

func cleanupComposeProject(taskCtx context.Context, svcCtx *svc.ServiceContext, projectID string, deleteDir bool) error {
	root, err := composeProject.FindProjectRoot(svcCtx, projectID)
	if err != nil {
		return err
	}
	projects, err := composeProject.ScanProjects(taskCtx, svcCtx)
	if err != nil {
		return err
	}
	found := false
	for _, project := range projects.Projects {
		if project.ID == projectID {
			found = true
			if project.Status != "unused" {
				return fmt.Errorf("项目状态已变化，不能清理")
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("项目状态已变化或项目不存在")
	}
	if err := taskCtx.Err(); err != nil {
		return err
	}
	if deleteDir {
		if err := composeProject.RemoveProjectRoot(root, svcCtx.Config.Compose.ScanPaths); err != nil {
			return err
		}
		if _, err := os.Lstat(root); err == nil {
			return fmt.Errorf("删除项目目录失败")
		} else if !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := taskCtx.Err(); err != nil {
			return err
		}
		if entry.IsDir() || (!composeProject.IsComposeFile(entry.Name()) && !composeProject.IsComposeEnvFile(entry.Name())) {
			continue
		}
		if err := os.Remove(filepath.Join(root, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
