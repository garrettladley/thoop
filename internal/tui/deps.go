package tui

import (
	"context"
	"log/slog"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/repository"
	"github.com/garrettladley/thoop/internal/xsync"
)

type Deps struct {
	Ctx          context.Context
	Cancel       context.CancelFunc
	Logger       *slog.Logger
	TokenChecker oauth.TokenChecker
	TokenSource  *oauth.DBTokenSource
	AuthFlow     oauth.Flow
	WhoopClient  *whoop.Client
	Repository   *repository.Repository
	SyncService  xsync.SyncService
	DataFetcher  xsync.DataFetcher
}
