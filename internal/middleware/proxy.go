package middleware

import "net/http"

// ProxyHeaders ignores forwarded client headers unless the proxy is trusted
func ProxyHeaders(trustProxy bool, next http.Handler) http.Handler {
	if trustProxy {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("Forwarded")
		r.Header.Del("X-Forwarded-For")
		r.Header.Del("X-Forwarded-Host")
		r.Header.Del("X-Forwarded-Proto")
		r.Header.Del("X-Real-IP")
		next.ServeHTTP(w, r)
	})
}
