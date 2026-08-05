package middleware

import "net/http"

// SecurityHeaders 为所有响应附加基础安全头（对 SPA + 同源 API 友好）。
func SecurityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		// 同源脚本/样式/图片；API 与 SPA 同域。不强制 upgrade-insecure（本地 HTTP 部署常见）。
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next(w, r)
	}
}
