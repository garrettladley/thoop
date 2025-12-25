package commands

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/garrettladley/thoop/internal/oauth"
)

func CheckAuthCmd(checker oauth.TokenChecker) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		hasToken, err := checker.HasToken(ctx)
		return AuthStatusMsg{HasToken: hasToken, Err: err}
	}
}
