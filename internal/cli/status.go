package cli

import (
	"fmt"

	"github.com/mucan54/envs3/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current project state",
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, cfg, _, pub, err := loadContext()
		if err != nil {
			return err
		}

		state, _ := config.LoadLocalState(cfg.Project)

		result, err := eng.Status(ctx(), pub)
		if err != nil {
			return err
		}

		fmt.Printf("Project:      %s\n", result.Project.Name)
		fmt.Printf("Storage:      s3 (%s)\n", cfg.Bucket)
		if state != nil && state.ActiveEnvironment != "" {
			es := state.Environments[state.ActiveEnvironment]
			fmt.Printf("Active env:   %s (v%d)\n", state.ActiveEnvironment, es.Version)
			if es.LastPull != "" {
				fmt.Printf("Last pull:    %s\n", es.LastPull)
			}
		}
		fmt.Println()

		fmt.Println("Environments:")
		for _, env := range result.Environments {
			access := "✗ no access"
			if env.HasAccess {
				access = "✓ access"
			}
			fmt.Printf("  %-15s v%-4d %3d keys  %-20s %s\n",
				env.Name, env.Version, env.KeyCount, env.UpdatedAt, access)
		}

		return nil
	},
}
