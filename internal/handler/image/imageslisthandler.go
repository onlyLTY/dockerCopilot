package image

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/logic/image"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ImagesListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := image.NewImagesListLogic(r.Context(), svcCtx)
		resp, err := l.ImagesList()
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
		} else {
			httpx.WriteJson(w, resp.Code, resp)
		}
	}
}
