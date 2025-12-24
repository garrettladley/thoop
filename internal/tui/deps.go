package tui

import (
	"context"

	"github.com/garrettladley/thoop/internal/client/whoop"
)

type TokenChecker interface {
	HasToken(ctx context.Context) (bool, error)
}

type Deps struct {
	TokenChecker TokenChecker
	WhoopClient  *whoop.Client
}
