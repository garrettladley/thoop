package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/garrettladley/thoop/internal/client/whoop"
)

func checkAuthCmd(checker TokenChecker) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		hasToken, err := checker.HasToken(ctx)
		return AuthStatusMsg{HasToken: hasToken, Err: err}
	}
}

func fetchMetricsCmd(client *whoop.Client) tea.Cmd {
	if client == nil {
		return func() tea.Msg {
			return MetricsDataMsg{Err: nil}
		}
	}

	return func() tea.Msg {
		var (
			ctx = context.Background()
			msg = MetricsDataMsg{}
		)

		cycles, err := client.Cycle.List(ctx, &whoop.ListParams{Limit: 1})
		if err != nil {
			msg.Err = err
			return msg
		}

		if len(cycles.Records) == 0 {
			return msg
		}

		cycle := cycles.Records[0]

		if cycle.Score != nil {
			msg.Strain = &cycle.Score.Strain
		}

		recovery, err := client.Cycle.GetRecovery(ctx, cycle.ID)
		if err == nil && recovery.Score != nil {
			msg.Recovery = &recovery.Score.RecoveryScore
		}

		sleep, err := client.Cycle.GetSleep(ctx, cycle.ID)
		if err == nil && sleep.Score != nil {
			msg.Sleep = &sleep.Score.SleepPerformancePercentage
		}

		return msg
	}
}
