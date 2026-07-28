package compose

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/compose"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ProjectsListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logic := compose.NewProjectsListLogic(r.Context(), svcCtx)
		resp, err := logic.List()
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

func PortsListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logic := compose.NewPortsListLogic(r.Context(), svcCtx)
		resp, err := logic.List()
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
