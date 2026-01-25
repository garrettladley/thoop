package tui

import (
	"context"
	"log/slog"

	"github.com/garrettladley/thoop/internal/cache"
	"github.com/garrettladley/thoop/internal/client/sse"
	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xsync"
)

type Deps struct {
	Ctx              context.Context
	Cancel           context.CancelFunc
	Logger           *slog.Logger
	TokenChecker     oauth.TokenChecker
	TokenSource      oauth.TokenSource
	AuthFlow         oauth.Flow
	CacheService     cache.CacheService
	SSEClient        *sse.Client
	NotifProcessor   *xsync.NotificationProcessor
	NotificationChan chan storage.Notification
}
