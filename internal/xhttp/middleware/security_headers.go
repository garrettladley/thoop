package middleware

import (
	"net/http"

	"github.com/garrettladley/thoop/internal/xhttp"
)

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(xhttp.XContentTypeOpts, "nosniff")
		w.Header().Set(xhttp.XFrameOpts, "DENY")
		w.Header().Set(xhttp.XXSSProtection, "1; mode=block")
		w.Header().Set(xhttp.ReferrerPolicy, "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
