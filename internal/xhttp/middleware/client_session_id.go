package middleware

import (
	"net/http"

	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xhttp"
)

func ClientSessionID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := xhttp.GetRequestHeaderSessionID(r)
		ctx := xcontext.SetSessionID(r.Context(), sessionID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
