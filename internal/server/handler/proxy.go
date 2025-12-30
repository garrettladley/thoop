package handler

import (
	"errors"
	"net/http"

	"github.com/garrettladley/thoop/internal/service/proxy"
	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xerrors"
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
		xerrors.WriteError(ctx, w, xerrors.Unauthorized(xerrors.WithMessage("missing user context")))
		return
	}

	info, err := h.service.CheckRateLimit(ctx, userID)
	if err != nil {
		if errors.Is(err, proxy.ErrRateLimited) && info != nil {
			xerrors.WriteError(ctx, w, xerrors.TooManyRequests(xerrors.WithMessage(info.Message), xerrors.WithRetryAfter(info.RetryAfter), xerrors.WithReason(info.Reason)))
			return
		}
		logger.ErrorContext(ctx, "failed to check rate limit",
			xslog.Error(err),
			xslog.UserID(userID))
		xerrors.WriteError(ctx, w, xerrors.Internal(xerrors.WithMessage("failed to check rate limit"), xerrors.WithCause(err)))
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
			xerrors.WriteError(ctx, w, xerrors.BadRequest(xerrors.WithMessage("invalid path")))
			return
		}
		logger.ErrorContext(ctx, "failed to proxy request",
			xslog.Error(err),
			xslog.UserID(userID))
		xerrors.WriteError(ctx, w, xerrors.Internal(xerrors.WithMessage("failed to proxy request"), xerrors.WithCause(err)))
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
