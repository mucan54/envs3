package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(rollbackCmd)
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback [environment] <version>",
	Short: "Restore a previous version",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, cfg, priv, pub, err := loadWriteContext()
		if err != nil {
			return err
		}

		var env string
		var versionStr string

		if len(args) == 1 {
			env = resolveEnv(cfg, "")
			versionStr = args[0]
		} else {
			env = args[0]
			versionStr = args[1]
		}

		// Strip 'v' prefix if present
		versionStr = strconv.Itoa(0) // reset
		if len(args) == 1 {
			versionStr = args[0]
		} else {
			versionStr = args[1]
		}
		if len(versionStr) > 0 && versionStr[0] == 'v' {
			versionStr = versionStr[1:]
		}

		version, err := strconv.Atoi(versionStr)
		if err != nil {
			return fmt.Errorf("invalid version: %s", versionStr)
		}

		email := getUserEmail()

		if !confirm(fmt.Sprintf("Rollback %s to v%d?", env, version)) {
			fmt.Println("Cancelled")
			return nil
		}

		result, err := eng.Rollback(ctx(), env, email, version, priv, pub)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Rolled back %s v%d → v%d (restored from v%d)\n", env, result.OldVersion, result.NewVersion, version)
		return nil
	},
}
