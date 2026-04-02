package cli

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/mucan54/envs3/internal/config"
	"github.com/mucan54/envs3/internal/crypto"
	"github.com/mucan54/envs3/internal/engine"
	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
	"github.com/spf13/cobra"
)

var (
	syncToken  string
	syncUser   string
	syncOutput string
)

func init() {
	syncCmd.Flags().StringVar(&syncToken, "token", "", "Service token (alternative to ENVS3_TOKEN env var)")
	syncCmd.Flags().StringVar(&syncUser, "user", "", "Set .env file owner (e.g., www-data)")
	syncCmd.Flags().StringVar(&syncOutput, "output", ".env", "Output file path")
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync secrets using a token (no local config needed)",
	Long: `Fetch and decrypt secrets using a service token.
Unlike 'pull', this command requires no local config files
(.env.envs3 or private key). Everything is in the token.

Examples:
  envs3 sync --token=$TOKEN
  envs3 sync --token=$TOKEN --user=www-data
  ENVS3_TOKEN=xxx envs3 sync --user=www-data`,
	RunE: func(cmd *cobra.Command, args []string) error {
		token := syncToken
		if token == "" {
			token = os.Getenv("ENVS3_TOKEN")
		}
		if token == "" {
			return fmt.Errorf("token required: use --token flag or ENVS3_TOKEN env var")
		}

		payload, err := config.ParseServiceToken(token)
		if err != nil {
			return err
		}

		// Check expiry
		if payload.ExpiresAt != nil {
			if exp, err := time.Parse(time.RFC3339, *payload.ExpiresAt); err == nil {
				if time.Now().UTC().After(exp) {
					return fmt.Errorf("token expired at %s. Request a new token", *payload.ExpiresAt)
				}
			}
		}

		store := storage.NewS3Store(storage.S3Config{
			Endpoint:        payload.Storage.Endpoint,
			Region:          payload.Storage.Region,
			Bucket:          payload.Storage.Bucket,
			AccessKeyID:     payload.Storage.AccessKeyID,
			SecretAccessKey: payload.Storage.SecretAccessKey,
		})

		eng := engine.NewEngine(store, payload.Project)

		priv, err := crypto.DecodeKey(payload.PrivateKey)
		if err != nil {
			return fmt.Errorf("invalid token: %w", err)
		}
		pub, err := crypto.DecodeKey(payload.PublicKey)
		if err != nil {
			return fmt.Errorf("invalid token: %w", err)
		}

		result, err := eng.Pull(ctx(), payload.Env, priv, pub, "")
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		format.WriteDotEnv(&buf, result.Secrets, result.Environment, result.Version, result.UpdatedAt)

		fileMode := os.FileMode(0644)
		if syncUser != "" {
			fileMode = 0600
		}

		if err := os.WriteFile(syncOutput, buf.Bytes(), fileMode); err != nil {
			return fmt.Errorf("write %s: %w", syncOutput, err)
		}

		if syncUser != "" {
			if err := chownFile(syncOutput, syncUser); err != nil {
				return fmt.Errorf("chown: %w", err)
			}
		}

		fmt.Printf("✓ Synced %s v%d (%d keys)\n", payload.Env, result.Version, len(result.Secrets))

		// Audit
		eng.AuditPull(ctx(), payload.Env, payload.Name, crypto.Fingerprint(pub), result.Version)

		return nil
	},
}
