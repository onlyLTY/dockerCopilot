package icons

import (
	"errors"
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/icons"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func UploadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			writeUploadError(w, http.StatusBadRequest, "failed to parse form")
			return
		}
		file, handler, err := r.FormFile("file")
		if err != nil {
			writeUploadError(w, http.StatusBadRequest, "failed to get file")
			return
		}
		defer file.Close()

		l := icons.NewUploadLogic(r.Context(), svcCtx)
		resp, err := l.Upload(&icons.UploadInput{
			ImageName:    r.FormValue("imageName"),
			OriginalName: handler.Filename,
			File:         file,
		})
		if err != nil {
			var ue *icons.UploadError
			if errors.As(err, &ue) {
				writeUploadError(w, ue.Status, ue.Msg)
				return
			}
			writeUploadError(w, http.StatusInternalServerError, err.Error())
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

func writeUploadError(w http.ResponseWriter, statusCode int, msg string) {
	httpx.WriteJson(w, statusCode, types.Resp{Code: statusCode, Msg: msg, Data: map[string]interface{}{}})
}
