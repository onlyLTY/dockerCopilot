package middleware

import (
	"net/http"
)

type IndexCheckMiddleware struct {
	Uuid string
}

func NewIndexCheckMiddleware(uuid string) *IndexCheckMiddleware {
	return &IndexCheckMiddleware{Uuid: uuid}
}

func (m *IndexCheckMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookies, err := r.Cookie("device_verified")
		if err != nil {
			next(w, r)
			return
		}
		token, err := validateJwtToken(m.Uuid, cookies.Value)
		if err != nil {
			next(w, r)
			return
		}
		if !token {
			next(w, r)
			return
		}
		w.Header().Set("Location", "/containersManager")
		w.WriteHeader(301)
	}
}
