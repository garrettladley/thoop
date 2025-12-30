package handler

import "net/http"

// HandleHealth returns a simple health check response.
func HandleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
