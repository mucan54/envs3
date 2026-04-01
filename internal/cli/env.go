package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var envFrom string

func init() {
	envCreateCmd.Flags().StringVar(&envFrom, "from", "", "Copy secrets from an existing environment")
	envCmd.AddCommand(envCreateCmd)
	envCmd.AddCommand(envListCmd)
	rootCmd.AddCommand(envCmd)
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environments",
}

var envCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, cfg, priv, pub, err := loadWriteContext()
		if err != nil {
			return err
		}
		_ = cfg

		email := getUserEmail()

		if err := eng.CreateEnv(ctx(), args[0], email, envFrom, priv, pub); err != nil {
			return err
		}

		fmt.Printf("✓ Environment '%s' created\n", args[0])
		if envFrom != "" {
			fmt.Printf("  Copied secrets from '%s'\n", envFrom)
		}
		return nil
	},
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List environments",
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, _, _, _, err := loadContext()
		if err != nil {
			return err
		}

		envs, err := eng.ListEnvs(ctx())
		if err != nil {
			return err
		}

		for _, e := range envs {
			fmt.Println(e)
		}
		return nil
	},
}
