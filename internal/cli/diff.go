package cli

import (
	"fmt"
	"os"

	"github.com/mucan54/envs3/internal/engine"
	"github.com/mucan54/envs3/internal/format"
	"github.com/spf13/cobra"
)

var diffAll bool

func init() {
	diffCmd.Flags().BoolVar(&diffAll, "all", false, "Show identical keys too")
	rootCmd.AddCommand(diffCmd)
}

var diffCmd = &cobra.Command{
	Use:   "diff [env1] [env2]",
	Short: "Compare environments or local .env vs remote",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, cfg, priv, pub, err := loadContext()
		if err != nil {
			return err
		}

		switch len(args) {
		case 0:
			// Diff local .env vs remote active environment
			env := resolveEnv(cfg, "")
			f, err := os.Open(".env")
			if err != nil {
				return fmt.Errorf("read .env: %w", err)
			}
			defer f.Close()
			local, err := format.ParseDotEnv(f)
			if err != nil {
				return err
			}
			entries, err := eng.DiffLocal(ctx(), env, local, priv, pub)
			if err != nil {
				return err
			}
			printDiff("local", env, entries)

		case 1:
			// Diff local .env vs named environment
			env := args[0]
			f, err := os.Open(".env")
			if err != nil {
				return fmt.Errorf("read .env: %w", err)
			}
			defer f.Close()
			local, err := format.ParseDotEnv(f)
			if err != nil {
				return err
			}
			entries, err := eng.DiffLocal(ctx(), env, local, priv, pub)
			if err != nil {
				return err
			}
			printDiff("local", env, entries)

		case 2:
			// Diff two remote environments
			entries, err := eng.Diff(ctx(), args[0], args[1], priv, pub)
			if err != nil {
				return err
			}
			printDiff(args[0], args[1], entries)
		}

		return nil
	},
}

func printDiff(leftName, rightName string, entries []engine.DiffEntry) {
	fmt.Printf("%-30s %-20s %-20s\n", "Key", leftName, rightName)
	fmt.Printf("%-30s %-20s %-20s\n", "---", "-----", "-----")

	hasDiff := false
	for _, e := range entries {
		if e.Status == "identical" && !diffAll {
			continue
		}
		hasDiff = true

		left := e.Left
		right := e.Right
		if left == "" {
			left = "✗ (not set)"
		}
		if right == "" {
			right = "✗ (not set)"
		}

		// Truncate long values
		if len(left) > 18 {
			left = left[:15] + "..."
		}
		if len(right) > 18 {
			right = right[:15] + "..."
		}

		suffix := ""
		if e.Status == "identical" {
			suffix = " (identical)"
		}
		fmt.Printf("%-30s %-20s %-20s%s\n", e.Key, left, right, suffix)
	}

	if !hasDiff {
		fmt.Println("No differences found")
	}
}
