package middleware

import (
	"net/http"

	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/google/uuid"
)

type RequestIDMiddleware struct {
	IDFunc func(*http.Request) string
}

var defaultRequestIDMiddleware = &RequestIDMiddleware{
	IDFunc: func(_ *http.Request) string {
		return uuid.New().String()
	},
}

type RequestIDOption func(*RequestIDMiddleware)

func RequestID(opts ...RequestIDOption) func(http.Handler) http.Handler {
	middleware := defaultRequestIDMiddleware

	for _, opt := range opts {
		opt(middleware)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := middleware.IDFunc(r)
			ctx := xcontext.SetRequestID(r.Context(), id)
			xhttp.SetHeaderRequestID(w, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
