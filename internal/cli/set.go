package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	setEnv   string
	setUnset string
	setYes   bool
	setForce bool
)

func init() {
	setCmd.Flags().StringVar(&setEnv, "env", "", "Target environment")
	setCmd.Flags().StringVar(&setUnset, "unset", "", "Comma-separated keys to remove")
	setCmd.Flags().BoolVar(&setYes, "yes", false, "Skip confirmation prompt")
	setCmd.Flags().BoolVar(&setForce, "force", false, "Force skip confirmation for non-default environments")
	rootCmd.AddCommand(setCmd)
}

var setCmd = &cobra.Command{
	Use:   "set KEY=VALUE [KEY=VALUE...]",
	Short: "Set key-value pairs on a remote environment",
	Args:  cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, cfg, priv, pub, err := loadWriteContext()
		if err != nil {
			return err
		}

		env := resolveEnv(cfg, setEnv)

		// Parse KEY=VALUE pairs
		setKVs := make(map[string]string)
		for _, arg := range args {
			idx := strings.IndexByte(arg, '=')
			if idx < 0 {
				return fmt.Errorf("invalid argument: %s (expected KEY=VALUE)", arg)
			}
			setKVs[arg[:idx]] = arg[idx+1:]
		}

		// Parse --unset keys
		var unsetKeys []string
		if setUnset != "" {
			for _, k := range strings.Split(setUnset, ",") {
				k = strings.TrimSpace(k)
				if k != "" {
					unsetKeys = append(unsetKeys, k)
				}
			}
		}

		if len(setKVs) == 0 && len(unsetKeys) == 0 {
			return fmt.Errorf("nothing to set or unset")
		}

		// Confirmation
		isDefault := (env == cfg.DefaultEnv)
		if !setYes || (!isDefault && !setForce) {
			fmt.Printf("  Target: %s\n\n", strings.ToUpper(env))
			fmt.Println("  Changes:")
			for k, v := range setKVs {
				fmt.Printf("    + %s = %s\n", k, v)
			}
			for _, k := range unsetKeys {
				fmt.Printf("    - %s (removed)\n", k)
			}
			fmt.Println()

			if isDefault {
				if !confirm(fmt.Sprintf("Apply %d change(s) to %s?", len(setKVs)+len(unsetKeys), env)) {
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
		}

		email := getUserEmail()

		result, err := eng.Set(ctx(), env, email, setKVs, unsetKeys, priv, pub)
		if err != nil {
			return err
		}

		if result == nil {
			fmt.Println("Nothing to change")
			return nil
		}

		fmt.Printf("✓ %s v%d → v%d\n", env, result.OldVersion, result.NewVersion)
		fmt.Println("\n  Local .env unchanged.")
		return nil
	},
}
