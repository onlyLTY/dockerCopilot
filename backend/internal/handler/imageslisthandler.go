package handler

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func imagesListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewImagesListLogic(r.Context(), svcCtx)
		resp, err := l.ImagesList()
		WriteLogicResp(w, r, resp, err)
	}
}
