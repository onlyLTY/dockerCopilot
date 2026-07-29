package compose

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/logic/compose"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func CreateProjectHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		var req types.ComposeProjectCreateReq
		var err error
		contentType := strings.ToLower(r.Header.Get("Content-Type"))
		if strings.HasPrefix(contentType, "application/json") {
			err = json.NewDecoder(r.Body).Decode(&req)
		} else {
			err = r.ParseForm()
			if err == nil {
				req.ProjectName = r.FormValue("projectName")
				req.Filename = r.FormValue("filename")
				req.Content = r.FormValue("content")
			}
		}
		if err != nil {
			auditError("create", r, err, started)
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := compose.NewFilesLogic(r.Context(), svcCtx).Create(&req)
		audit("create", r, resp, started)
		writeResponse(r, w, resp, err)
	}
}

func ListFilesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return fileRequestHandler(svcCtx, "file_list", func(l *compose.FilesLogic, req *types.ComposeProjectFileReq) (*types.Resp, error) {
		return l.List(req)
	})
}

func ReadFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return fileRequestHandler(svcCtx, "file_read", func(l *compose.FilesLogic, req *types.ComposeProjectFileReq) (*types.Resp, error) {
		return l.Read(req)
	})
}

func fileRequestHandler(svcCtx *svc.ServiceContext, operation string, fn func(*compose.FilesLogic, *types.ComposeProjectFileReq) (*types.Resp, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		var req types.ComposeProjectFileReq
		if err := httpx.Parse(r, &req); err != nil {
			auditError(operation, r, err, started)
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := fn(compose.NewFilesLogic(r.Context(), svcCtx), &req)
		audit(operation, r, resp, started)
		writeResponse(r, w, resp, err)
	}
}

func UpdateFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		var req types.ComposeProjectFileUpdateReq
		if err := httpx.Parse(r, &req); err != nil {
			auditError("file_update", r, err, started)
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := compose.NewFilesLogic(r.Context(), svcCtx).Update(&req)
		audit("file_update", r, resp, started)
		writeResponse(r, w, resp, err)
	}
}

func ValidateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		var req types.ComposeProjectValidateReq
		if err := httpx.Parse(r, &req); err != nil {
			auditError("validate", r, err, started)
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := compose.NewFilesLogic(r.Context(), svcCtx).Validate(&req)
		audit("validate", r, resp, started)
		writeResponse(r, w, resp, err)
	}
}

func writeResponse(r *http.Request, w http.ResponseWriter, resp *types.Resp, err error) {
	if err != nil {
		httpx.WriteJson(w, resp.Code, resp)
		return
	}
	httpx.OkJsonCtx(r.Context(), w, resp)
}
