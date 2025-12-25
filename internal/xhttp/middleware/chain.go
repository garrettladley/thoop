package middleware

import (
	"net/http"
	"slices"
)

func Chain(h http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	slices.Reverse(middleware)
	for _, m := range middleware {
		h = m(h)
	}
	return h
}
