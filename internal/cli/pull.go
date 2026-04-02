package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/mucan54/envs3/internal/config"
	"github.com/mucan54/envs3/internal/format"
	"github.com/spf13/cobra"
)

var (
	pullOutput  string
	pullForce   bool
	pullJSONMeta bool
	pullNoHook  bool
)

func init() {
	pullCmd.Flags().StringVar(&pullOutput, "output", ".env", "Write to custom path")
	pullCmd.Flags().BoolVar(&pullForce, "force", false, "Skip ETag check")
	pullCmd.Flags().BoolVar(&pullJSONMeta, "json-meta", false, "Write metadata to stdout as JSON")
	pullCmd.Flags().BoolVar(&pullNoHook, "no-hook", false, "Skip post_pull hook")
	rootCmd.AddCommand(pullCmd)
}

var pullCmd = &cobra.Command{
	Use:   "pull [environment]",
	Short: "Fetch and decrypt secrets, writing to .env",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, cfg, priv, pub, err := loadContext()
		if err != nil {
			return err
		}

		env := resolveEnv(cfg, "")
		if len(args) > 0 {
			env = args[0]
		}

		state, err := config.LoadLocalState(cfg.Project)
		if err != nil {
			return err
		}

		// Check if switching environments
		if state.ActiveEnvironment != "" && state.ActiveEnvironment != env {
			// Backup current .env
			if _, err := os.Stat(pullOutput); err == nil {
				data, _ := os.ReadFile(pullOutput)
				if len(data) > 0 {
					os.WriteFile(pullOutput+".previous", data, 0644)
				}
			}
		}

		cachedETag := ""
		if !pullForce {
			if es, ok := state.Environments[env]; ok {
				cachedETag = es.ETag
			}
		}

		result, err := eng.Pull(ctx(), env, priv, pub, cachedETag)
		if err != nil {
			return err
		}

		if result.UpToDate {
			es := state.Environments[env]
			fmt.Printf("✓ Already up to date (%s v%d)\n", env, es.Version)
			return nil
		}

		// Write .env
		var buf bytes.Buffer
		format.WriteDotEnv(&buf, result.Secrets, result.Environment, result.Version, result.UpdatedAt)
		if err := os.WriteFile(pullOutput, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("write %s: %w", pullOutput, err)
		}

		// Compute change summary
		oldKeys := make(map[string]bool)
		if es, ok := state.Environments[env]; ok {
			for _, k := range es.KnownKeys {
				oldKeys[k] = true
			}
		}

		newKeys := make([]string, 0, len(result.Secrets))
		for k := range result.Secrets {
			newKeys = append(newKeys, k)
		}
		sort.Strings(newKeys)

		// Update state
		state.ActiveEnvironment = env
		state.Environments[env] = format.EnvState{
			ETag:      result.ETag,
			Version:   result.Version,
			LastPull:  result.UpdatedAt,
			KnownKeys: newKeys,
		}
		config.SaveLocalState(cfg.Project, state)

		// Print summary
		oldVersion := 0
		if es, ok := state.Environments[env]; ok {
			oldVersion = es.Version
		}
		if oldVersion > 0 && oldVersion != result.Version {
			fmt.Printf("✓ Updated %s v%d → v%d\n", env, oldVersion, result.Version)
		} else {
			fmt.Printf("✓ Pulled %s v%d (%d keys)\n", env, result.Version, len(result.Secrets))
		}

		for _, k := range newKeys {
			if !oldKeys[k] {
				fmt.Printf("  + %s added\n", k)
			}
		}
		for k := range oldKeys {
			if _, ok := result.Secrets[k]; !ok {
				fmt.Printf("  - %s removed\n", k)
			}
		}

		// Expiry & rotation warnings (OWASP compliance)
		if result.Metadata != nil {
			now := time.Now().UTC()
			for key, meta := range result.Metadata {
				if meta.ExpiresAt != "" {
					if exp, err := time.Parse(time.RFC3339, meta.ExpiresAt); err == nil {
						if now.After(exp) {
							fmt.Printf("  ✗ %s EXPIRED (since %s)\n", key, meta.ExpiresAt[:10])
						} else if exp.Sub(now).Hours() < 7*24 {
							fmt.Printf("  ⚠ %s expires in %d days\n", key, int(exp.Sub(now).Hours()/24))
						}
					}
				}
				if meta.RotationDays > 0 && meta.RotatedAt != "" {
					if rot, err := time.Parse(time.RFC3339, meta.RotatedAt); err == nil {
						daysSince := int(now.Sub(rot).Hours() / 24)
						if daysSince > meta.RotationDays {
							fmt.Printf("  ✗ %s rotation overdue (%d days, policy: %d days)\n", key, daysSince, meta.RotationDays)
						}
					}
				}
			}
		}

		// JSON metadata output
		if pullJSONMeta {
			meta := map[string]interface{}{
				"environment": result.Environment,
				"version":     result.Version,
				"count":       len(result.Secrets),
				"keys":        newKeys,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.Encode(meta)
		}

		// Post-pull hook
		if !pullNoHook && cfg.HookPostPull != "" {
			fmt.Printf("Running post_pull hook: %s\n", cfg.HookPostPull)
			// Hook execution would go here
		}

		return nil
	},
}
