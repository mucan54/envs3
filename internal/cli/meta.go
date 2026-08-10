package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mucan54/envs3/internal/format"
	"github.com/spf13/cobra"
)

var (
	metaEnv          string
	metaRotationDays int
	metaExpiresAt    string
	metaTags         string
	metaDescription  string
)

func init() {
	metaSetCmd.Flags().StringVar(&metaEnv, "env", "", "Target environment")
	metaSetCmd.Flags().IntVar(&metaRotationDays, "rotation-days", 0, "Rotation policy in days (e.g., 90)")
	metaSetCmd.Flags().StringVar(&metaExpiresAt, "expires-at", "", "Expiry date (ISO 8601 or days from now, e.g., '2026-12-31' or '90d')")
	metaSetCmd.Flags().StringVar(&metaTags, "tags", "", "Comma-separated tags (e.g., 'database,critical,pci')")
	metaSetCmd.Flags().StringVar(&metaDescription, "description", "", "Human-readable description")

	metaGetCmd.Flags().StringVar(&metaEnv, "env", "", "Target environment")

	metaCmd.AddCommand(metaSetCmd)
	metaCmd.AddCommand(metaGetCmd)
	rootCmd.AddCommand(metaCmd)
}

var metaCmd = &cobra.Command{
	Use:   "meta",
	Short: "Manage secret metadata (rotation policy, expiry, tags)",
}

var metaSetCmd = &cobra.Command{
	Use:   "set <KEY>",
	Short: "Set metadata for a secret key",
	Long: `Set OWASP-compliant metadata for a secret key.

Examples:
  envs3 meta set DB_PASSWORD --rotation-days=90
  envs3 meta set API_KEY --expires-at=2026-12-31
  envs3 meta set STRIPE_KEY --tags=payment,critical --description="Stripe live key"
  envs3 meta set DB_PASSWORD --rotation-days=90 --env=production`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, cfg, priv, pub, err := loadWriteContext()
		if err != nil {
			return err
		}

		env := resolveEnv(cfg, metaEnv)
		key := args[0]
		email := getUserEmail()

		meta := format.SecretMetadata{
			RotationDays: metaRotationDays,
			Description:  metaDescription,
		}

		if metaRotationDays > 0 {
			// Set rotated_at to now if setting rotation policy for first time
			meta.RotatedAt = time.Now().UTC().Format(time.RFC3339)
		}

		if metaExpiresAt != "" {
			meta.ExpiresAt = parseExpiryInput(metaExpiresAt)
		}

		if metaTags != "" {
			for _, tag := range strings.Split(metaTags, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					meta.Tags = append(meta.Tags, tag)
				}
			}
		}

		if err := eng.SetSecretMetadata(ctx(), env, email, key, meta, priv, pub); err != nil {
			return err
		}

		fmt.Printf("✓ Metadata updated for %s in %s\n", key, env)
		if metaRotationDays > 0 {
			fmt.Printf("  Rotation policy: every %d days\n", metaRotationDays)
		}
		if meta.ExpiresAt != "" {
			fmt.Printf("  Expires at: %s\n", meta.ExpiresAt)
		}
		if len(meta.Tags) > 0 {
			fmt.Printf("  Tags: %s\n", strings.Join(meta.Tags, ", "))
		}

		return nil
	},
}

var metaGetCmd = &cobra.Command{
	Use:   "get <KEY>",
	Short: "Show metadata for a secret key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, cfg, priv, pub, err := loadContext()
		if err != nil {
			return err
		}

		env := resolveEnv(cfg, metaEnv)
		result, err := eng.View(ctx(), env, priv, pub)
		if err != nil {
			return err
		}

		key := args[0]
		if _, exists := result.Secrets[key]; !exists {
			return fmt.Errorf("key '%s' not found in %s", key, env)
		}

		// We need to read bundle directly for metadata
		var bundle format.Bundle
		eng.Store.Get(ctx(), "")
		_ = bundle

		fmt.Printf("Key: %s\n", key)
		fmt.Printf("Environment: %s\n", env)
		// Metadata display would go here once we can read it
		fmt.Println("(Use 'envs3 compliance' to check all metadata)")

		return nil
	},
}

func parseExpiryInput(input string) string {
	// Try ISO 8601 date
	if _, err := time.Parse("2006-01-02", input); err == nil {
		return input + "T00:00:00Z"
	}
	if _, err := time.Parse(time.RFC3339, input); err == nil {
		return input
	}

	// Try "Nd" format (days from now)
	if strings.HasSuffix(input, "d") {
		if days, err := strconv.Atoi(input[:len(input)-1]); err == nil {
			return time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour).Format(time.RFC3339)
		}
	}

	return input
}
