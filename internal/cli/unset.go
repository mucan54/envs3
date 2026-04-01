package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var unsetEnv string

func init() {
	unsetCmd.Flags().StringVar(&unsetEnv, "env", "", "Target environment")
	rootCmd.AddCommand(unsetCmd)
}

var unsetCmd = &cobra.Command{
	Use:   "unset KEY [KEY...]",
	Short: "Remove keys from a remote environment",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, cfg, priv, pub, err := loadWriteContext()
		if err != nil {
			return err
		}

		env := resolveEnv(cfg, unsetEnv)

		// Confirmation
		isDefault := (env == cfg.Defaults.Environment)
		fmt.Printf("  Target: %s\n\n", strings.ToUpper(env))
		fmt.Println("  Changes:")
		for _, k := range args {
			fmt.Printf("    - %s (removed)\n", k)
		}
		fmt.Println()

		if isDefault {
			if !confirm(fmt.Sprintf("Remove %d key(s) from %s?", len(args), env)) {
				fmt.Println("Cancelled")
				return nil
			}
		} else {
			answer := prompt(fmt.Sprintf("Type '%s' to confirm: ", env))
			if answer != env {
				fmt.Println("Cancelled")
				return nil
			}
		}

		email := getUserEmail()

		result, err := eng.Set(ctx(), env, email, nil, args, priv, pub)
		if err != nil {
			return err
		}

		if result == nil {
			fmt.Println("Nothing to change")
			return nil
		}

		fmt.Printf("✓ %s v%d → v%d\n", env, result.OldVersion, result.NewVersion)
		return nil
	},
}
