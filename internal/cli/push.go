package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/mucan54/envs3/internal/config"
	"github.com/mucan54/envs3/internal/format"
	"github.com/spf13/cobra"
)

var (
	pushYes     bool
	pushMessage string
)

func init() {
	pushCmd.Flags().BoolVar(&pushYes, "yes", false, "Skip confirmation prompt")
	pushCmd.Flags().StringVar(&pushMessage, "message", "", "Attach a message to the version")
	rootCmd.AddCommand(pushCmd)
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push local .env changes to the active environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, cfg, priv, pub, err := loadWriteContext()
		if err != nil {
			return err
		}

		state, err := config.LoadLocalState(cfg.Project)
		if err != nil {
			return err
		}

		if state.ActiveEnvironment == "" {
			return fmt.Errorf("no active environment. Run 'envs3 pull <env>' first, then push")
		}

		env := state.ActiveEnvironment

		// Read local .env
		f, err := os.Open(".env")
		if err != nil {
			return fmt.Errorf("read .env: %w", err)
		}
		defer f.Close()

		localSecrets, err := format.ParseDotEnv(f)
		if err != nil {
			return fmt.Errorf("parse .env: %w", err)
		}

		email := getUserEmail()

		result, err := eng.Push(ctx(), env, email, localSecrets, priv, pub)
		if err != nil {
			return err
		}

		if result == nil {
			fmt.Println("Nothing to push")
			return nil
		}

		// Show changes
		if !pushYes {
			fmt.Printf("Changes to push (%s):\n", env)
			sort.Slice(result.Changes, func(i, j int) bool {
				return result.Changes[i].Key < result.Changes[j].Key
			})
			for _, c := range result.Changes {
				switch c.Action {
				case "added":
					fmt.Printf("  + %s (new)\n", c.Key)
				case "updated":
					fmt.Printf("  △ %s changed\n", c.Key)
				case "removed":
					fmt.Printf("  - %s (removed)\n", c.Key)
				}
			}
			if !confirm(fmt.Sprintf("Push %d changes?", len(result.Changes))) {
				fmt.Println("Cancelled")
				return nil
			}
		}

		// Update local state
		es := state.Environments[env]
		es.Version = result.NewVersion
		keys := make([]string, 0, len(localSecrets))
		for k := range localSecrets {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		es.KnownKeys = keys
		state.Environments[env] = es
		config.SaveLocalState(cfg.Project, state)

		fmt.Printf("✓ Pushed %s v%d → v%d\n", env, result.OldVersion, result.NewVersion)
		return nil
	},
}
