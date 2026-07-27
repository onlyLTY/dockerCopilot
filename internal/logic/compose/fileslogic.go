package compose

import (
	"context"
	"fmt"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	composeProject "github.com/onlyLTY/dockerCopilot/internal/utiles/compose_project"
	"github.com/zeromicro/go-zero/core/logx"
)

type FilesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFilesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilesLogic {
	return &FilesLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *FilesLogic) Create(req *types.ComposeProjectCreateReq) (*types.Resp, error) {
	resp := &types.Resp{}
	if err := l.validateContent(req.Content); err != nil {
		return errorResp(resp, 400, err.Error()), nil
	}
	id, version, err := composeProject.CreateProject(l.svcCtx, req.ProjectName, req.Filename, req.Content)
	if err != nil {
		l.Errorf("create compose project: %v", err)
		return errorResp(resp, 400, err.Error()), nil
	}
	return successResp(resp, map[string]interface{}{"projectId": id, "version": version}), nil
}

func (l *FilesLogic) List(req *types.ComposeProjectFileReq) (*types.Resp, error) {
	resp := &types.Resp{}
	root, err := composeProject.FindProjectRoot(l.svcCtx, req.ProjectID)
	if err != nil {
		return errorResp(resp, 404, err.Error()), nil
	}
	files, err := composeProject.ListProjectFiles(root)
	if err != nil {
		return errorResp(resp, 500, "读取 Compose 文件失败"), err
	}
	return successResp(resp, files), nil
}

func (l *FilesLogic) Read(req *types.ComposeProjectFileReq) (*types.Resp, error) {
	resp := &types.Resp{}
	root, err := composeProject.FindProjectRoot(l.svcCtx, req.ProjectID)
	if err != nil {
		return errorResp(resp, 404, err.Error()), nil
	}
	content, version, err := composeProject.ReadProjectFile(root, req.Filename)
	if err != nil {
		return errorResp(resp, 404, "读取 Compose 文件失败"), nil
	}
	return successResp(resp, map[string]interface{}{"filename": req.Filename, "content": string(content), "version": version}), nil
}

func (l *FilesLogic) Update(req *types.ComposeProjectFileUpdateReq) (*types.Resp, error) {
	resp := &types.Resp{}
	if err := l.validateContent(req.Content); err != nil {
		return errorResp(resp, 400, err.Error()), nil
	}
	root, err := composeProject.FindProjectRoot(l.svcCtx, req.ProjectID)
	if err != nil {
		return errorResp(resp, 404, err.Error()), nil
	}
	version, err := composeProject.SaveProjectFile(l.svcCtx, root, req.Filename, req.Content, req.Version)
	if err != nil {
		l.Errorf("update compose file: %v", err)
		return errorResp(resp, 409, err.Error()), nil
	}
	return successResp(resp, map[string]interface{}{"version": version}), nil
}

func (l *FilesLogic) Validate(req *types.ComposeProjectValidateReq) (*types.Resp, error) {
	resp := &types.Resp{}
	root := ""
	var err error
	if req.ProjectID != "" {
		root, err = composeProject.FindProjectRoot(l.svcCtx, req.ProjectID)
		if err != nil {
			return errorResp(resp, 404, err.Error()), nil
		}
	} else if len(l.svcCtx.Config.Compose.ScanPaths) > 0 {
		root = l.svcCtx.Config.Compose.ScanPaths[0]
	} else {
		return errorResp(resp, 400, "未配置 Compose 扫描目录"), nil
	}
	if err := l.validateContent(req.Content); err != nil {
		return errorResp(resp, 400, err.Error()), nil
	}
	project, err := composeProject.ParseComposeContent(root, req.Filename, []byte(req.Content))
	if err != nil {
		return errorResp(resp, 400, err.Error()), nil
	}
	return successResp(resp, map[string]interface{}{"valid": true, "name": project.Name, "services": project.ServiceNames()}), nil
}

func (l *FilesLogic) validateContent(content string) error {
	if content == "" {
		return fmt.Errorf("Compose 文件不能为空")
	}
	if limit := l.svcCtx.Config.Compose.MaxFileSize; limit > 0 && int64(len(content)) > limit {
		return fmt.Errorf("Compose 文件超过大小限制")
	}
	return nil
}

func successResp(resp *types.Resp, data interface{}) *types.Resp {
	resp.Code, resp.Msg, resp.Data = 200, "success", data
	return resp
}

func errorResp(resp *types.Resp, code int, msg string) *types.Resp {
	resp.Code, resp.Msg, resp.Data = code, msg, map[string]interface{}{}
	return resp
}
