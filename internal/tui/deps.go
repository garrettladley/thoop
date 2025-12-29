package tui

import (
	"context"
	"log/slog"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/oauth"
)

type Deps struct {
	Ctx          context.Context
	Cancel       context.CancelFunc
	Logger       *slog.Logger
	TokenChecker oauth.TokenChecker
	TokenSource  *oauth.DBTokenSource
	AuthFlow     oauth.Flow
	WhoopClient  *whoop.Client
}
