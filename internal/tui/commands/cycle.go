package commands

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/garrettladley/thoop/internal/client/whoop"
)

func FetchCycleCmd(ctx context.Context, client *whoop.Client) tea.Cmd {
	if client == nil {
		return func() tea.Msg {
			return CycleMsg{Cycle: nil}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()
		cycles, err := client.Cycle.List(ctx, &whoop.ListParams{Limit: 1})
		if err != nil {
			return CycleMsg{Err: err}
		}
		if len(cycles.Records) == 0 {
			return CycleMsg{Cycle: nil}
		}
		return CycleMsg{Cycle: &cycles.Records[0]}
	}
}
