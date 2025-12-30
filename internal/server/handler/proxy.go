package handler

import (
	"errors"
	"net/http"

	"github.com/garrettladley/thoop/internal/apperr"
	"github.com/garrettladley/thoop/internal/service/proxy"
	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xslog"
)

type Proxy struct {
	service proxy.Service
}

func NewProxy(service proxy.Service) *Proxy {
	return &Proxy{service: service}
}

// HandleWhoopProxy handles requests to /api/whoop/*.
func (h *Proxy) HandleWhoopProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := xslog.FromContext(ctx)

	userID, ok := xcontext.GetWhoopUserID(ctx)
	if !ok || userID == 0 {
		logger.WarnContext(ctx, "missing user key in context")
		apperr.WriteError(w, apperr.Unauthorized("unauthorized", "missing user context"))
		return
	}

	info, err := h.service.CheckRateLimit(ctx, userID)
	if err != nil {
		if errors.Is(err, proxy.ErrRateLimited) && info != nil {
			apperr.WriteError(w, apperr.TooManyRequests("rate_limited", info.Message, info.RetryAfter, info.Reason))
			return
		}
		logger.ErrorContext(ctx, "failed to check rate limit",
			xslog.Error(err),
			xslog.UserID(userID))
		apperr.WriteError(w, apperr.Internal("internal_error", "failed to check rate limit", err))
		return
	}

	proxyReq := &proxy.ProxyRequest{
		Method:  r.Method,
		Path:    r.URL.Path,
		Query:   r.URL.RawQuery,
		Headers: r.Header,
		Body:    r.Body,
		UserID:  userID,
	}

	resp, err := h.service.ProxyRequest(ctx, proxyReq)
	if err != nil {
		if errors.Is(err, proxy.ErrInvalidPath) {
			apperr.WriteError(w, apperr.BadRequest("invalid_request", "invalid path"))
			return
		}
		logger.ErrorContext(ctx, "failed to proxy request",
			xslog.Error(err),
			xslog.UserID(userID))
		apperr.WriteError(w, apperr.Internal("internal_error", "failed to proxy request", err))
		return
	}

	logger.InfoContext(ctx, "proxied request to WHOOP API",
		xslog.RequestMethod(r),
		xslog.RequestPath(r),
		xslog.HTTPStatus(resp.StatusCode),
		xslog.UserID(userID))

	if err := proxy.CopyResponse(w, resp); err != nil {
		logger.ErrorContext(ctx, "failed to copy response body", xslog.Error(err))
	}
}
