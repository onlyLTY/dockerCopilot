package imagesManager

import (
	"net/http"

	"github.com/onlyLTY/oneKeyUpdate/v2/internal/logic/imagesManager"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetNewImageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetNewImageReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := imagesManager.NewGetNewImageLogic(r.Context(), svcCtx)
		resp, err := l.GetNewImage(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
