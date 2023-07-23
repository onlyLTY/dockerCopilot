package imagesManager

import (
	"github.com/flosch/pongo2"
	"github.com/zeromicro/go-zero/core/logx"
	"net/http"

	"github.com/onlyLTY/oneKeyUpdate/v2/internal/logic/imagesManager"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ImagesManagerIndexHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := imagesManager.NewImagesManagerIndexLogic(r.Context(), svcCtx)
		list, err := l.ImagesManagerIndex()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			t, err := svcCtx.Template.FromFile("templates/imagesManager/imagesManager.html")
			if err != nil {
				logx.Error(err)
			}
			execute, err := t.ExecuteBytes(pongo2.Context{"images_list": list})
			if err != nil {
				logx.Error(err)
			}
			w.Write(execute)
			httpx.Ok(w)
		}
	}
}
