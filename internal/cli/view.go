package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	viewEnv    string
	viewKeys   bool
	viewKey    string
	viewFormat string
)

func init() {
	viewCmd.Flags().StringVar(&viewEnv, "env", "", "Target environment")
	viewCmd.Flags().BoolVar(&viewKeys, "keys", false, "Show only key names")
	viewCmd.Flags().StringVar(&viewKey, "key", "", "Show value of a single key")
	viewCmd.Flags().StringVar(&viewFormat, "format", "dotenv", "Output format: dotenv, json, table")
	rootCmd.AddCommand(viewCmd)
}

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "Display environment secrets without modifying files",
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, cfg, priv, pub, err := loadContext()
		if err != nil {
			return err
		}

		env := resolveEnv(cfg, viewEnv)

		// Keys-only mode (no DEK needed)
		if viewKeys {
			keys, _, err := eng.ViewKeys(ctx(), env)
			if err != nil {
				return err
			}
			for _, k := range keys {
				fmt.Println(k)
			}
			return nil
		}

		result, err := eng.View(ctx(), env, priv, pub)
		if err != nil {
			return err
		}

		// Single key mode
		if viewKey != "" {
			val, ok := result.Secrets[viewKey]
			if !ok {
				return fmt.Errorf("key '%s' not found in %s", viewKey, env)
			}
			fmt.Print(val)
			return nil
		}

		switch viewFormat {
		case "json":
			out := map[string]interface{}{
				"environment": result.Environment,
				"version":     result.Version,
				"secrets":     result.Secrets,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(out)

		case "table":
			fmt.Printf("%-30s %s\n", "KEY", "VALUE")
			fmt.Printf("%-30s %s\n", "---", "-----")
			for _, k := range result.Keys {
				fmt.Printf("%-30s %s\n", k, result.Secrets[k])
			}

		default: // dotenv
			for _, k := range result.Keys {
				fmt.Printf("%s=%s\n", k, result.Secrets[k])
			}
		}

		return nil
	},
}
