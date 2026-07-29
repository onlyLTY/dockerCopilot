package compose

import (
	"net/http"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/logic/compose"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ProjectsListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		resp, err := compose.NewProjectsListLogic(r.Context(), svcCtx).List()
		audit("project_list", r, resp, started)
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

func PortsListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		resp, err := compose.NewPortsListLogic(r.Context(), svcCtx).List()
		audit("ports_list", r, resp, started)
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
