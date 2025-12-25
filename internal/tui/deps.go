package tui

import (
	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/oauth"
)

type Deps struct {
	TokenChecker oauth.TokenChecker
	WhoopClient  *whoop.Client
}
