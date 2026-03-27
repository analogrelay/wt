package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "wt",
	Short: "Worktree manager with tmux session integration",
	Long: `wt manages git worktrees with tmux session integration.

Navigate between projects, create worktrees, and monitor
repository health — all with a polished terminal UI.`,
	Version: Version,
	// Default to status when no subcommand is given
	RunE: func(cmd *cobra.Command, args []string) error {
		return statusCmd.RunE(statusCmd, args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}
