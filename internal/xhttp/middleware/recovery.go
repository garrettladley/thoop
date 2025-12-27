package middleware

import (
	"net/http"

	"github.com/garrettladley/thoop/internal/xslog"
)

// Recovery handles panics and logs them with structured error groups.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				xslog.FromContext(r.Context()).ErrorContext(
					r.Context(),
					"panic recovered",
					xslog.RequestGroupMinimal(r),
					xslog.ErrorGroupWithStack(err),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
