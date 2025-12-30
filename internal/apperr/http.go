package apperr

import (
	"errors"
	"net/http"

	"github.com/garrettladley/thoop/internal/xhttp"
	go_json "github.com/goccy/go-json"
)

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func WriteError(w http.ResponseWriter, err error) {
	var appErr *Error
	if !errors.As(err, &appErr) {
		var rlErr *RateLimitError
		if errors.As(err, &rlErr) {
			writeRateLimitError(w, rlErr)
			return
		}
		appErr = Internal("internal_error", "an unexpected error occurred", err)
	}

	xhttp.SetHeaderContentTypeApplicationJSON(w)
	w.WriteHeader(appErr.StatusCode)
	_ = go_json.NewEncoder(w).Encode(errorResponse{
		Error:   appErr.Code,
		Message: appErr.Message,
	})
}

func writeRateLimitError(w http.ResponseWriter, err *RateLimitError) {
	xhttp.SetHeaderContentTypeApplicationJSON(w)
	xhttp.SetHeaderRetryAfter(w, err.RetryAfter)
	if err.Reason != "" {
		w.Header().Set(xhttp.XRateLimitReason, err.Reason)
	}
	w.WriteHeader(http.StatusTooManyRequests)
	_ = go_json.NewEncoder(w).Encode(errorResponse{
		Error:   err.Code,
		Message: err.Message,
	})
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	xhttp.SetHeaderContentTypeApplicationJSON(w)
	w.WriteHeader(status)
	_ = go_json.NewEncoder(w).Encode(data)
}

func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func WriteOK(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, data)
}
