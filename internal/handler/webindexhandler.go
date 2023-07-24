package handler

import (
	"net/http"

	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
)

func webindexHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/containersManager")
		w.WriteHeader(301)
	}
}
