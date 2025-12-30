//go:build !release

package main

import "github.com/spf13/cobra"

func addDevCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(authCmd())
	rootCmd.AddCommand(testCmd())
}
