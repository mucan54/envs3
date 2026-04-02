package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/mucan54/envs3/internal/format"
	"github.com/spf13/cobra"
)

var (
	auditEnv    string
	auditActor  string
	auditAction string
	auditLimit  int
)

func init() {
	auditCmd.Flags().StringVar(&auditEnv, "env", "", "Filter by environment")
	auditCmd.Flags().StringVar(&auditActor, "actor", "", "Filter by actor (email)")
	auditCmd.Flags().StringVar(&auditAction, "action", "", "Filter by action (pull, push, set, member_add, etc.)")
	auditCmd.Flags().IntVar(&auditLimit, "limit", 50, "Maximum entries to show")
	rootCmd.AddCommand(auditCmd)
}

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View audit log — who did what, when",
	Long: `View the audit trail of all operations performed on this project.

Every pull, push, set, unset, member add/remove, token create/revoke,
and rollback operation is logged to S3.

Examples:
  envs3 audit                          # Show recent audit entries
  envs3 audit --env=production         # Filter by environment
  envs3 audit --actor=can@company.com  # Filter by actor
  envs3 audit --action=push            # Filter by action type
  envs3 audit --limit=100              # Show more entries`,
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, _, _, _, err := loadContext()
		if err != nil {
			return err
		}

		entries, err := eng.ListAuditLog(ctx(), auditEnv, auditActor, auditAction, auditLimit)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Println("No audit entries found")
			return nil
		}

		fmt.Printf("%-20s %-12s %-25s %-12s %s\n", "Timestamp", "Action", "Actor", "Environment", "Details")
		fmt.Printf("%-20s %-12s %-25s %-12s %s\n", "─────────", "──────", "─────", "───────────", "───────")

		for _, e := range entries {
			ts := e.Timestamp
			if t, err := time.Parse(time.RFC3339, e.Timestamp); err == nil {
				ts = t.Format("2006-01-02 15:04:05")
			}

			details := formatAuditDetails(e)
			fmt.Printf("%-20s %-12s %-25s %-12s %s\n", ts, e.Action, e.Actor, e.Environment, details)
		}

		return nil
	},
}

func formatAuditDetails(e format.AuditEntry) string {
	if e.Details == nil {
		return ""
	}
	d := e.Details

	switch e.Action {
	case "pull":
		return fmt.Sprintf("v%d", d.ToVersion)
	case "push", "set":
		parts := []string{}
		if len(d.KeysAdded) > 0 {
			parts = append(parts, fmt.Sprintf("+%d", len(d.KeysAdded)))
		}
		if len(d.KeysUpdated) > 0 {
			parts = append(parts, fmt.Sprintf("~%d", len(d.KeysUpdated)))
		}
		if len(d.KeysRemoved) > 0 {
			parts = append(parts, fmt.Sprintf("-%d", len(d.KeysRemoved)))
		}
		version := ""
		if d.FromVersion > 0 {
			version = fmt.Sprintf("v%d→v%d ", d.FromVersion, d.ToVersion)
		}
		return version + strings.Join(parts, " ")
	case "member_add":
		return fmt.Sprintf("%s (%s)", d.MemberEmail, strings.Join(d.Environments, ","))
	case "member_remove":
		return fmt.Sprintf("%s", d.MemberEmail)
	case "token_create", "token_revoke":
		return d.TokenName
	default:
		return ""
	}
}
