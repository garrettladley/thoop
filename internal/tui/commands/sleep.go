package commands

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/garrettladley/thoop/internal/client/whoop"
)

func FetchSleepCmd(ctx context.Context, client *whoop.Client, cycleID int64) tea.Cmd {
	if client == nil {
		return func() tea.Msg {
			return SleepMsg{Sleep: nil}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()
		sleep, err := client.Cycle.GetSleep(ctx, cycleID)
		return SleepMsg{Sleep: sleep, Err: err}
	}
}
