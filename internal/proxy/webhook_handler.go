package proxy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xslog"
	go_json "github.com/goccy/go-json"
)

const (
	headerWhoopSignature          = "X-Whoop-Signature"
	headerWhoopSignatureTimestamp = "X-Whoop-Signature-Timestamp"

	maxWebhookAge = 5 * time.Minute
)

type WhoopWebhookEvent interface {
	webhookEvent()
	GetUserID() int64
	GetAction() storage.Action
	GetEntityType() storage.EntityType
	GetEntityID() string
}

type webhookEventBase struct {
	Type   string `json:"type"`
	UserID int64  `json:"user_id"`
}

func (e webhookEventBase) GetUserID() int64 { return e.UserID }

type WorkoutWebhookEvent struct {
	webhookEventBase
	WorkoutID string `json:"id"`
	action    storage.Action
}

func (e WorkoutWebhookEvent) webhookEvent()                     {}
func (e WorkoutWebhookEvent) GetAction() storage.Action         { return e.action }
func (e WorkoutWebhookEvent) GetEntityType() storage.EntityType { return storage.EntityTypeWorkout }
func (e WorkoutWebhookEvent) GetEntityID() string               { return e.WorkoutID }

type SleepWebhookEvent struct {
	webhookEventBase
	SleepID string `json:"id"`
	action  storage.Action
}

func (e SleepWebhookEvent) webhookEvent()                     {}
func (e SleepWebhookEvent) GetAction() storage.Action         { return e.action }
func (e SleepWebhookEvent) GetEntityType() storage.EntityType { return storage.EntityTypeSleep }
func (e SleepWebhookEvent) GetEntityID() string               { return e.SleepID }

type RecoveryWebhookEvent struct {
	webhookEventBase
	SleepID string `json:"id"`
	action  storage.Action
}

func (e RecoveryWebhookEvent) webhookEvent()                     {}
func (e RecoveryWebhookEvent) GetAction() storage.Action         { return e.action }
func (e RecoveryWebhookEvent) GetEntityType() storage.EntityType { return storage.EntityTypeRecovery }
func (e RecoveryWebhookEvent) GetEntityID() string               { return e.SleepID }

type rawWebhookPayload struct {
	Type   string `json:"type"`
	UserID int64  `json:"user_id"`
	ID     string `json:"id"`
}

func parseWebhookEvent(data []byte) (WhoopWebhookEvent, error) {
	var raw rawWebhookPayload
	if err := go_json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	parts := strings.SplitN(raw.Type, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid event type format: %s", raw.Type)
	}

	entity, actionStr := parts[0], parts[1]

	var action storage.Action
	switch actionStr {
	case "updated":
		action = storage.ActionUpdated
	case "deleted":
		action = storage.ActionDeleted
	default:
		return nil, fmt.Errorf("unknown action: %s", actionStr)
	}

	base := webhookEventBase{Type: raw.Type, UserID: raw.UserID}

	switch entity {
	case "workout":
		return WorkoutWebhookEvent{webhookEventBase: base, WorkoutID: raw.ID, action: action}, nil
	case "sleep":
		return SleepWebhookEvent{webhookEventBase: base, SleepID: raw.ID, action: action}, nil
	case "recovery":
		return RecoveryWebhookEvent{webhookEventBase: base, SleepID: raw.ID, action: action}, nil
	default:
		return nil, fmt.Errorf("unknown entity type: %s", entity)
	}
}

type WebhookHandler struct {
	clientSecret      string
	notificationStore storage.NotificationStore
}

func NewWebhookHandler(clientSecret string, notificationStore storage.NotificationStore) *WebhookHandler {
	return &WebhookHandler{
		clientSecret:      clientSecret,
		notificationStore: notificationStore,
	}
}

func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := xslog.FromContext(ctx)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.ErrorContext(ctx, "failed to read webhook body", xslog.Error(err))
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	signature := r.Header.Get(headerWhoopSignature)
	timestamp := r.Header.Get(headerWhoopSignatureTimestamp)

	if signature == "" || timestamp == "" {
		logger.WarnContext(ctx, "missing webhook signature headers")
		http.Error(w, "missing signature headers", http.StatusUnauthorized)
		return
	}

	if !h.verifySignature(body, timestamp, signature) {
		logger.WarnContext(ctx, "invalid webhook signature")
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	if !h.isTimestampValid(timestamp) {
		logger.WarnContext(ctx, "webhook timestamp too old", xslog.Timestamp(timestamp))
		http.Error(w, "timestamp too old", http.StatusUnauthorized)
		return
	}

	event, err := parseWebhookEvent(body)
	if err != nil {
		logger.WarnContext(ctx, "unknown webhook event", xslog.Error(err))
		w.WriteHeader(http.StatusOK)
		return
	}

	notification := storage.Notification{
		EntityType: event.GetEntityType(),
		EntityID:   event.GetEntityID(),
		Action:     event.GetAction(),
		Timestamp:  time.Now(),
	}

	userID := event.GetUserID()

	if err := h.notificationStore.Add(ctx, userID, notification); err != nil {
		logger.ErrorContext(ctx, "failed to store notification",
			xslog.Error(err),
			xslog.UserID(userID),
		)
	}

	if err := h.notificationStore.Publish(ctx, userID, notification); err != nil {
		logger.ErrorContext(ctx, "failed to publish notification",
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

	w.WriteHeader(http.StatusOK)
}

// verifySignature verifies the webhook signature using HMAC-SHA256.
// algorithm: base64(HMAC-SHA256(timestamp + body, client_secret))
func (h *WebhookHandler) verifySignature(body []byte, timestamp, signature string) bool {
	mac := hmac.New(sha256.New, []byte(h.clientSecret))
	mac.Write([]byte(timestamp))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

func (h *WebhookHandler) isTimestampValid(timestampStr string) bool {
	var timestampMs int64
	if _, err := fmt.Sscanf(timestampStr, "%d", &timestampMs); err != nil {
		return false
	}

	webhookTime := time.UnixMilli(timestampMs)
	return time.Since(webhookTime) <= maxWebhookAge
}
