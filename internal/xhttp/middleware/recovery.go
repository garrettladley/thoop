package middleware

import (
	"net/http"

	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xslog"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				xslog.FromContext(r.Context()).ErrorContext(
					r.Context(),
					"panic recovered",
					xslog.RequestGroup(r),
					xslog.ErrorGroupWithStack(err),
				)
				xhttp.Error(w, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
