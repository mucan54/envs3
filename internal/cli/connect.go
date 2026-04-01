package cli

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(connectCmd)
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to an existing project (alias for 'envs3 init')",
	RunE:  runInit,
}
