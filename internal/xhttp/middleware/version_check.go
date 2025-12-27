package middleware

import (
	"net/http"

	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/version"
	"github.com/garrettladley/thoop/internal/xslog"
	go_json "github.com/goccy/go-json"
)

func VersionCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := xslog.FromContext(r.Context())

		clientVersion := r.Header.Get(version.Header)
		if clientVersion == "" {
			clientVersion = "unknown"
		}

		if verr := version.CheckCompatibility(clientVersion); verr != nil {
			logger.WarnContext(
				r.Context(),
				"client version incompatible",
				xslog.ClientVersion(verr.ClientVersion),
				xslog.ProxyVersion(verr.ProxyVersion),
				xslog.MinVersion(verr.MinVersion),
				xslog.RequestPath(r),
			)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUpgradeRequired)

			if err := go_json.NewEncoder(w).Encode(map[string]any{
				"error":       string(oauth.ErrorCodeIncompatibleVersion),
				"message":     verr.Error(),
				"min_version": verr.MinVersion,
			}); err != nil {
				logger.ErrorContext(
					r.Context(),
					"failed to encode version error response",
					xslog.ErrorGroup(err),
					xslog.RequestPath(r),
				)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}
