package xerrors

import (
	"net/http"

	"github.com/garrettladley/thoop/internal/xhttp"
	go_json "github.com/goccy/go-json"
)

type errorResponse struct {
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func WriteError(w http.ResponseWriter, err error) {
	appErr := As(err)
	if appErr == nil {
		appErr = Internal(WithCause(err))
	}

	xhttp.SetHeaderContentTypeApplicationJSON(w)

	if appErr.RateLimit != nil {
		if appErr.RateLimit.RetryAfter > 0 {
			xhttp.SetHeaderRetryAfter(w, appErr.RateLimit.RetryAfter)
		}
		if appErr.RateLimit.Reason != "" {
			w.Header().Set(xhttp.XRateLimitReason, appErr.RateLimit.Reason)
		}
	}

	w.WriteHeader(appErr.StatusCode)

	resp := errorResponse{Message: appErr.Message}
	if appErr.Validation != nil {
		resp.Fields = appErr.Validation.Fields
	}

	_ = go_json.NewEncoder(w).Encode(resp)
}
