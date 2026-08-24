package container

import (
	"mime"
	"net/http"
	"strconv"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func DownloadBackupHandler(_ *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		content, filename, err := utiles.ReadBackupDownload(r.URL.Query().Get("filename"))
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, types.Resp{
				Code: http.StatusBadRequest, Msg: "无法下载备份文件", Data: map[string]interface{}{},
			})
			return
		}
		contentDisposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
		w.Header().Set("Content-Disposition", contentDisposition)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}
}
