package compose

import (
	"net/http"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/logic/compose"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

// ComposeProjectsHandler goctl 风格：Logic → 统一写响应 + audit。
func ComposeProjectsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		l := compose.NewComposeProjectsLogic(r.Context(), svcCtx)
		resp, err := l.ComposeProjects()
		audit("project_list", r, resp, started)
		writeLogicResp(w, r, resp, err)
	}
}
