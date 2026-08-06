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

func ComposeProjectCreateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
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
		resp, err := compose.NewComposeProjectCreateLogic(r.Context(), svcCtx).ComposeProjectCreate(&req)
		audit("create", r, resp, started)
		writeLogicResp(w, r, resp, err)
	}
}
