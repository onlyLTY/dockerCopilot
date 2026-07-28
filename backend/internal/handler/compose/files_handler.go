package compose

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/onlyLTY/dockerCopilot/internal/logic/compose"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func CreateProjectHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ComposeProjectCreateReq
		contentType := strings.ToLower(r.Header.Get("Content-Type"))
		var err error
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
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := compose.NewFilesLogic(r.Context(), svcCtx).Create(&req)
		writeResponse(r, w, resp, err)
	}
}

func ListFilesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ComposeProjectFileReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := compose.NewFilesLogic(r.Context(), svcCtx).List(&req)
		writeResponse(r, w, resp, err)
	}
}

func ReadFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ComposeProjectFileReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := compose.NewFilesLogic(r.Context(), svcCtx).Read(&req)
		writeResponse(r, w, resp, err)
	}
}

func UpdateFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ComposeProjectFileUpdateReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := compose.NewFilesLogic(r.Context(), svcCtx).Update(&req)
		writeResponse(r, w, resp, err)
	}
}

func ValidateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ComposeProjectValidateReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := compose.NewFilesLogic(r.Context(), svcCtx).Validate(&req)
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
