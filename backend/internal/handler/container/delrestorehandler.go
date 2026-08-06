package container

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/onlyLTY/dockerCopilot/internal/logic/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
)

func DelRestoreHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DelContainerBackupReq
		if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") && r.Body != nil {
			var body struct {
				Filename string `json:"filename"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil && body.Filename != "" {
				req.Filename = body.Filename
			}
		}
		if req.Filename == "" {
			req.Filename = r.URL.Query().Get("filename")
		}
		if req.Filename == "" {
			req.Filename = r.FormValue("filename")
		}

		l := container.NewDelRestoreLogic(r.Context(), svcCtx)
		resp, err := l.DelRestore(&req)
		writeLogicResp(w, r, resp, err)
	}
}
