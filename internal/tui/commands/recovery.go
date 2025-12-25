package commands

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/garrettladley/thoop/internal/client/whoop"
)

func FetchRecoveryCmd(ctx context.Context, client *whoop.Client, cycleID int64) tea.Cmd {
	if client == nil {
		return func() tea.Msg {
			return RecoveryMsg{Recovery: nil}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()
		recovery, err := client.Cycle.GetRecovery(ctx, cycleID)
		return RecoveryMsg{Recovery: recovery, Err: err}
	}
}
