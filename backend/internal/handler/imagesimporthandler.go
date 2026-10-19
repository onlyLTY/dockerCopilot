package handler

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

// ImportImagesHandler 接收 docker save 导出的镜像包（原始请求体流式读取，
// 不经过 multipart，避免大镜像包占用内存）。上传完成后立即返回 taskID，
// docker load 在后台任务中执行，可前往任务页面查看进度或停止任务。
func ImportImagesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := svcCtx.Config.ImageImportLimit()
		// MaxBytesReader 同时约束 Content-Length 与实际读取字节数（含 chunked 上传）。
		r.Body = http.MaxBytesReader(w, r.Body, limit)

		l := logic.NewImportImagesLogic(r.Context(), svcCtx)
		resp, err := l.SubmitImport(logic.ImportInput{
			Filename: r.URL.Query().Get("filename"),
			TaskID:   r.URL.Query().Get("taskID"),
			Reader:   r.Body,
			Total:    r.ContentLength,
		})
		WriteLogicResp(w, r, resp, err)
	}
}
