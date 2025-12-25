package commands

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/garrettladley/thoop/internal/oauth"
)

func CheckAuthCmd(ctx context.Context, checker oauth.TokenChecker) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		hasToken, err := checker.HasToken(ctx)
		return AuthStatusMsg{HasToken: hasToken, Err: err}
	}
}
