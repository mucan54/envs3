package cli

import (
	"fmt"
	"time"

	"github.com/mucan54/envs3/internal/config"
	"github.com/mucan54/envs3/internal/storage"
	"github.com/spf13/cobra"
)

var (
	tokenCreateEnv      string
	tokenCreateName     string
	tokenCreateReadonly bool
	tokenCreateReadwrite bool
	tokenCreateTTL      string
)

func init() {
	tokenCreateCmd.Flags().StringVar(&tokenCreateEnv, "env", "", "Environment name")
	tokenCreateCmd.Flags().StringVar(&tokenCreateName, "name", "", "Token identifier")
	tokenCreateCmd.Flags().BoolVar(&tokenCreateReadonly, "readonly", true, "Read-only token")
	tokenCreateCmd.Flags().BoolVar(&tokenCreateReadwrite, "readwrite", false, "Read-write token")
	tokenCreateCmd.Flags().StringVar(&tokenCreateTTL, "ttl", "", "Time-to-live (e.g., 2h, 7d)")
	tokenCreateCmd.MarkFlagRequired("env")
	tokenCreateCmd.MarkFlagRequired("name")

	tokenCmd.AddCommand(tokenCreateCmd)
	tokenCmd.AddCommand(tokenListCmd)
	tokenCmd.AddCommand(tokenRevokeCmd)
	rootCmd.AddCommand(tokenCmd)
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage service tokens",
}

var tokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a service token for CI/CD",
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, cfg, priv, pub, err := loadWriteContext()
		if err != nil {
			return err
		}

		perm := "ro"
		if tokenCreateReadwrite {
			perm = "rw"
		}

		var ttl time.Duration
		if tokenCreateTTL != "" {
			ttl, err = time.ParseDuration(tokenCreateTTL)
			if err != nil {
				return fmt.Errorf("invalid TTL: %w", err)
			}
		}

		email := getUserEmail()

		// Load admin credentials for the S3 config to embed in token
		adminCreds, _ := config.LoadAdminCredentials(cfg.Project)
		s3Cfg := storage.S3Config{
			Endpoint:        cfg.Storage.Endpoint,
			Bucket:          cfg.Storage.Bucket,
			AccessKeyID:     cfg.Storage.ReadAccessKeyID,
			SecretAccessKey: cfg.Storage.ReadSecretAccessKey,
		}
		if perm == "rw" && adminCreds != nil {
			s3Cfg.AccessKeyID = adminCreds.WriteAccessKeyID
			s3Cfg.SecretAccessKey = adminCreds.WriteSecretAccessKey
		}

		token, err := eng.CreateToken(ctx(), tokenCreateName, tokenCreateEnv, perm, email, ttl, s3Cfg, priv, pub)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Token created: %s (%s, %s)\n\n", tokenCreateName, tokenCreateEnv, perm)
		fmt.Printf("ENVS3_TOKEN=%s\n\n", token)
		fmt.Println("Add this to your CI/CD secrets. This token will not be shown again.")

		return nil
	},
}

var tokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, _, _, _, err := loadContext()
		if err != nil {
			return err
		}

		tokens, err := eng.ListTokens(ctx())
		if err != nil {
			return err
		}

		if len(tokens) == 0 {
			fmt.Println("No tokens")
			return nil
		}

		fmt.Printf("%-20s %-15s %-10s %-25s %s\n", "Name", "Environment", "Perm", "Created", "Expires")
		for _, t := range tokens {
			expires := "never"
			if t.ExpiresAt != nil {
				expires = *t.ExpiresAt
			}
			fmt.Printf("%-20s %-15s %-10s %-25s %s\n",
				t.Name, t.Environment, t.Permissions, t.CreatedAt, expires)
		}
		return nil
	},
}

var tokenRevokeCmd = &cobra.Command{
	Use:   "revoke <name>",
	Short: "Revoke a service token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, _, priv, pub, err := loadWriteContext()
		if err != nil {
			return err
		}

		email := getUserEmail()

		if err := eng.RevokeToken(ctx(), args[0], priv, pub, email); err != nil {
			return err
		}

		fmt.Printf("✓ Token '%s' revoked. DEK rotated.\n", args[0])
		return nil
	},
}
