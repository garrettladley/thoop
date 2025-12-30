package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/garrettladley/thoop/internal/service/webhook"
	"github.com/garrettladley/thoop/internal/xerrors"
	"github.com/garrettladley/thoop/internal/xslog"
)

const (
	headerWhoopSignature          = "X-Whoop-Signature"
	headerWhoopSignatureTimestamp = "X-Whoop-Signature-Timestamp"
)

type Webhook struct {
	service webhook.Service
}

func NewWebhook(service webhook.Service) *Webhook {
	return &Webhook{service: service}
}

// HandleWebhook handles POST /webhooks/whoop requests.
func (h *Webhook) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := xslog.FromContext(ctx)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.ErrorContext(ctx, "failed to read webhook body", xslog.Error(err))
		xerrors.WriteError(w, xerrors.BadRequest(xerrors.WithMessage("failed to read request body")))
		return
	}

	req := webhook.ProcessRequest{
		Body:      body,
		Signature: r.Header.Get(headerWhoopSignature),
		Timestamp: r.Header.Get(headerWhoopSignatureTimestamp),
	}

	if err := h.service.ProcessWebhook(ctx, req); err != nil {
		// for unknown events, return 200 (WHOOP expects this)
		if errors.Is(err, webhook.ErrUnknownEventType) {
			logger.WarnContext(ctx, "unknown webhook event", xslog.Error(err))
			w.WriteHeader(http.StatusOK)
			return
		}

		if errors.Is(err, webhook.ErrMissingSignature) {
			logger.WarnContext(ctx, "missing webhook signature headers")
			xerrors.WriteError(w, xerrors.Unauthorized(xerrors.WithMessage("missing signature headers")))
			return
		}

		if errors.Is(err, webhook.ErrInvalidSignature) {
			logger.WarnContext(ctx, "invalid webhook signature")
			xerrors.WriteError(w, xerrors.Unauthorized(xerrors.WithMessage("invalid signature")))
			return
		}

		if errors.Is(err, webhook.ErrTimestampExpired) {
			logger.WarnContext(ctx, "webhook timestamp too old")
			xerrors.WriteError(w, xerrors.Unauthorized(xerrors.WithMessage("timestamp too old")))
			return
		}

		logger.ErrorContext(ctx, "failed to process webhook", xslog.Error(err))
		xerrors.WriteError(w, xerrors.Internal(xerrors.WithMessage("failed to process webhook"), xerrors.WithCause(err)))
		return
	}

	w.WriteHeader(http.StatusOK)
}
