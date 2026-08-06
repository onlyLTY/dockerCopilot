package container

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/onlyLTY/dockerCopilot/internal/logic/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func RenameHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ContainerRenameReq
		if err := httpx.ParsePath(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		if err := parseRenameBody(body, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := container.NewRenameLogic(r.Context(), svcCtx)
		resp, err := l.Rename(&req)
		writeLogicResp(w, r, resp, err)
	}
}

func parseRenameBody(body []byte, req *types.ContainerRenameReq) error {
	value := strings.TrimSpace(string(body))
	if strings.HasPrefix(value, "{") {
		if err := json.Unmarshal(body, req); err != nil {
			return err
		}
	} else if form, err := url.ParseQuery(value); err == nil {
		req.NewName = form.Get("newName")
	} else {
		return err
	}
	req.NewName = strings.TrimSpace(req.NewName)
	if req.NewName == "" {
		return errors.New(`field "newName" is not set`)
	}
	return nil
}
