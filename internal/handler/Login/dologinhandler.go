package Login

import (
	"github.com/flosch/pongo2"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/logic/Login"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"time"
)

func DoLoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DoLoginReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := Login.NewDoLoginLogic(r.Context(), svcCtx)
		err := l.DoLogin(&req)
		if err != nil {
			t, _ := svcCtx.Template.FromFile("templates/login/login.html")
			//if err != nil {
			//	logx.Error(err)
			//}
			execute, err := t.ExecuteBytes(pongo2.Context{"current_year": time.Now(), "error_message": err.Error()})
			if err != nil {
				logx.Error(err)
			}
			w.Write(execute)
			httpx.Ok(w)
		} else {
			w.Header().Set("Location", "/containersManager")
			w.WriteHeader(301)
		}
	}
}
