package handler

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func webindexHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/manager")
		w.WriteHeader(301)
	}
}
