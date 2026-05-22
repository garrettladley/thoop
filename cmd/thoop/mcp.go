package main

import (
	"fmt"

	"github.com/garrettladley/thoop/internal/mcpserver"
	"github.com/spf13/cobra"
)

func mcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run and connect the local thoop MCP server",
	}

	cmd.AddCommand(mcpServeCmd())
	cmd.AddCommand(mcpConnectCmd())

	return cmd
}

func mcpServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the local thoop MCP server over stdio",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mcpserver.RunStdio(cmd.Context())
		},
	}
}

func mcpConnectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "connect <claude|claude-code|claude-desktop|codex|codex-cli|codex-app|web>",
		Short: "Connect local thoop MCP to an MCP client",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := mcpserver.ParseConnectTarget(args[0])
			if err != nil {
				return fmt.Errorf("parse MCP connect target: %w", err)
			}
			results, err := mcpserver.Connect(cmd.Context(), target)
			if err != nil {
				return fmt.Errorf("connect MCP target: %w", err)
			}

			for _, result := range results {
				fmt.Printf("%s: %s\n", result.Target, result.Message)
				for _, changed := range result.Changed {
					fmt.Printf("  changed: %s\n", changed)
				}
			}

			return nil
		},
	}
}
