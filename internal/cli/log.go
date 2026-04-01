package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/mucan54/envs3/internal/format"
	"github.com/spf13/cobra"
)

var logLimit int

func init() {
	logCmd.Flags().IntVar(&logLimit, "limit", 20, "Show last n versions")
	rootCmd.AddCommand(logCmd)
}

var logCmd = &cobra.Command{
	Use:   "log [environment]",
	Short: "Show version history",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, cfg, priv, pub, err := loadContext()
		if err != nil {
			return err
		}
		_ = priv
		_ = pub

		env := resolveEnv(cfg, "")
		if len(args) > 0 {
			env = args[0]
		}

		entries, err := eng.Log(ctx(), env, logLimit)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Println("No history found")
			return nil
		}

		fmt.Printf("%-8s %-25s %-15s %s\n", "Version", "Author", "Date", "Changes")
		for _, e := range entries {
			date := e.CreatedAt
			if t, err := time.Parse(time.RFC3339, e.CreatedAt); err == nil {
				date = formatRelativeTime(t)
			}

			changeSummary := summarizeChanges(e.Changes)
			fmt.Printf("v%-7d %-25s %-15s %s\n", e.Version, e.CreatedBy, date, changeSummary)
		}

		return nil
	},
}

func formatRelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	}
}

func summarizeChanges(changes []format.Change) string {
	if len(changes) == 0 {
		return "(no changes recorded)"
	}
	if len(changes) <= 3 {
		parts := make([]string, len(changes))
		for i, c := range changes {
			parts[i] = fmt.Sprintf("%s %s", c.Key, c.Action)
		}
		return strings.Join(parts, ", ")
	}
	return fmt.Sprintf("%d keys changed", len(changes))
}
