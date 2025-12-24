package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/garrettladley/thoop/internal/tui"
	"github.com/spf13/cobra"
)

func tuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive TUI",
		Long:  "Opens the full-screen terminal UI for viewing WHOOP data.",
		RunE: func(cmd *cobra.Command, args []string) error {
			model := tui.New()

			p := tea.NewProgram(&model)

			if _, err := p.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}

			return nil
		},
	}
}
