package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var complianceEnv string

func init() {
	complianceCmd.Flags().StringVar(&complianceEnv, "env", "", "Check specific environment (default: all)")
	rootCmd.AddCommand(complianceCmd)
}

var complianceCmd = &cobra.Command{
	Use:   "compliance",
	Short: "Check OWASP secrets management compliance",
	Long: `Check your project against OWASP secrets management best practices.

Checks include:
  - Secret rotation policies and overdue rotations
  - Expired or soon-to-expire secrets
  - Token expiry policies
  - Audit log presence
  - Encryption verification (always passes — envs3 encrypts by design)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, cfg, priv, pub, err := loadContext()
		if err != nil {
			return err
		}

		// Determine environments to check
		envs := []string{resolveEnv(cfg, complianceEnv)}
		if complianceEnv == "" {
			// Check all environments the user has access to
			allEnvs, err := eng.ListEnvs(ctx())
			if err == nil && len(allEnvs) > 0 {
				envs = allEnvs
			}
		}

		fmt.Println("envs3 — OWASP Compliance Check")
		fmt.Println()

		// Static checks that always pass
		fmt.Println("✓ Encryption at rest (AES-256-GCM)")
		fmt.Println("✓ Encryption in transit (TLS via S3)")
		fmt.Println("✓ Client-side encryption (zero-knowledge)")
		fmt.Println("✓ Envelope encryption (X25519 + AES-256-GCM)")
		fmt.Println("✓ Per-environment access control (keyring-based)")
		fmt.Println()

		totalCritical := 0
		totalWarning := 0
		totalInfo := 0

		for _, env := range envs {
			issues, err := eng.CheckCompliance(ctx(), env, priv, pub)
			if err != nil {
				fmt.Printf("  ⚠ %s: %v\n", env, err)
				continue
			}

			for _, issue := range issues {
				icon := "ℹ"
				switch issue.Severity {
				case "critical":
					icon = "✗"
					totalCritical++
				case "warning":
					icon = "⚠"
					totalWarning++
				case "info":
					icon = "ℹ"
					totalInfo++
				}

				loc := issue.Environment
				if issue.Key != "" {
					loc += "/" + issue.Key
				}
				if loc == "" {
					loc = "project"
				}

				fmt.Printf("%s [%s] %s — %s\n", icon, strings.ToUpper(issue.Severity), loc, issue.Message)
			}
		}

		fmt.Println()
		if totalCritical > 0 {
			fmt.Printf("Results: %d critical, %d warnings, %d info\n", totalCritical, totalWarning, totalInfo)
		} else if totalWarning > 0 {
			fmt.Printf("Results: %d warnings, %d info (no critical issues)\n", totalWarning, totalInfo)
		} else {
			fmt.Println("✓ No critical or warning issues found")
		}

		// OWASP requirements that need cloud version
		fmt.Println()
		fmt.Println("Cloud version required for:")
		fmt.Println("  · Automatic scheduled rotation")
		fmt.Println("  · Runtime secret injection (API)")
		fmt.Println("  · Real-time expiry notifications")
		fmt.Println("  · Per-secret access policies")
		fmt.Println("  · Compliance certifications (SOC 2, HIPAA)")

		return nil
	},
}
