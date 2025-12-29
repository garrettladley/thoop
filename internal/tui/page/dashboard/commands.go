package dashboard

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/garrettladley/thoop/internal/client/whoop"
)

type CycleMsg struct {
	Cycle *whoop.Cycle
	Err   error
}

type SleepMsg struct {
	Sleep *whoop.Sleep
	Err   error
}

type RecoveryMsg struct {
	Recovery *whoop.Recovery
	Err      error
}

func FetchCycleCmd(ctx context.Context, client *whoop.Client) tea.Cmd {
	if client == nil {
		return func() tea.Msg {
			return CycleMsg{Cycle: nil}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
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

func FetchSleepCmd(ctx context.Context, client *whoop.Client, cycleID int64) tea.Cmd {
	if client == nil {
		return func() tea.Msg {
			return SleepMsg{Sleep: nil}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		sleep, err := client.Cycle.GetSleep(ctx, cycleID)
		return SleepMsg{Sleep: sleep, Err: err}
	}
}

func FetchRecoveryCmd(ctx context.Context, client *whoop.Client, cycleID int64) tea.Cmd {
	if client == nil {
		return func() tea.Msg {
			return RecoveryMsg{Recovery: nil}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		recovery, err := client.Cycle.GetRecovery(ctx, cycleID)
		return RecoveryMsg{Recovery: recovery, Err: err}
	}
}
