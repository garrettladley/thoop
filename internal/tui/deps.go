package tui

import (
	"context"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/oauth"
)

type Deps struct {
	Ctx          context.Context
	Cancel       context.CancelFunc
	TokenChecker oauth.TokenChecker
	WhoopClient  *whoop.Client
}
