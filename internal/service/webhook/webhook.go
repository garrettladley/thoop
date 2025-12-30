package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xslog"
)

const maxWebhookAge = 5 * time.Minute

type Processor struct {
	clientSecret      string
	notificationStore storage.NotificationStore
}

var _ Service = (*Processor)(nil)

func NewProcessor(clientSecret string, notificationStore storage.NotificationStore) *Processor {
	return &Processor{
		clientSecret:      clientSecret,
		notificationStore: notificationStore,
	}
}

func (p *Processor) ProcessWebhook(ctx context.Context, req ProcessRequest) error {
	logger := xslog.FromContext(ctx)

	if req.Signature == "" || req.Timestamp == "" {
		return ErrMissingSignature
	}

	if !p.verifySignature(req.Body, req.Timestamp, req.Signature) {
		return ErrInvalidSignature
	}

	if !p.isTimestampValid(req.Timestamp) {
		return ErrTimestampExpired
	}

	event, err := ParseEvent(req.Body)
	if err != nil {
		return err
	}

	notification := storage.Notification{
		TraceID:    event.GetTraceID(),
		EntityType: event.GetEntityType(),
		EntityID:   event.GetEntityID(),
		Action:     event.GetAction(),
		Timestamp:  time.Now(),
	}

	userID := event.GetUserID()

	if err := p.notificationStore.Add(ctx, userID, notification); err != nil {
		logger.ErrorContext(ctx, "failed to store/publish notification",
			xslog.Error(err),
			xslog.UserID(userID),
		)
	}

	logger.InfoContext(ctx, "processed webhook",
		xslog.EntityType(string(event.GetEntityType())),
		xslog.Action(string(event.GetAction())),
		xslog.UserID(userID),
		xslog.EntityID(event.GetEntityID()),
	)

	return nil
}

// verifySignature verifies the webhook signature using HMAC-SHA256.
// algorithm: base64(HMAC-SHA256(timestamp + body, client_secret))
func (p *Processor) verifySignature(body []byte, timestamp, signature string) bool {
	mac := hmac.New(sha256.New, []byte(p.clientSecret))
	mac.Write([]byte(timestamp))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

func (p *Processor) isTimestampValid(timestampStr string) bool {
	var timestampMs int64
	if _, err := fmt.Sscanf(timestampStr, "%d", &timestampMs); err != nil {
		return false
	}

	webhookTime := time.UnixMilli(timestampMs)
	return time.Since(webhookTime) <= maxWebhookAge
}
