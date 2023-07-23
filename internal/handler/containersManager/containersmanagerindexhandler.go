package containersManager

import (
	"github.com/flosch/pongo2"
	"github.com/zeromicro/go-zero/core/logx"
	"net/http"
	"time"

	"github.com/onlyLTY/oneKeyUpdate/v2/internal/logic/containersManager"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ContainersManagerIndexHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := containersManager.NewContainersManagerIndexLogic(r.Context(), svcCtx)
		err := l.ContainersManagerIndex()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			t, err := svcCtx.Template.FromFile("templates/containersManager/containersManager.html")
			if err != nil {
				logx.Error(err)
			}
			execute, err := t.ExecuteBytes(pongo2.Context{"current_year": time.Now()})
			if err != nil {
				logx.Error(err)
			}
			w.Write(execute)
			httpx.Ok(w)
		}
	}
}
