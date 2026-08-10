package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/mucan54/envs3/internal/config"
	"github.com/mucan54/envs3/internal/crypto"
	"github.com/mucan54/envs3/internal/engine"
	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
	"github.com/spf13/cobra"
)

var (
	runRestart  string
	runUser     string
	runInterval int
	runOutput   string
)

func init() {
	runCmd.Flags().StringVar(&runRestart, "restart", "", "Command to run after .env update (e.g., systemctl restart php-fpm)")
	runCmd.Flags().StringVar(&runUser, "user", "", "Set .env file owner (e.g., www-data)")
	runCmd.Flags().IntVar(&runInterval, "interval", 30, "Check interval in seconds")
	runCmd.Flags().StringVar(&runOutput, "output", ".env", "Output file path")
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Watch for secret changes and auto-update .env",
	Long: `Run as a daemon that watches S3 for secret changes.
When changes are detected, pulls the new secrets and optionally
restarts a service.

Requires ENVS3_TOKEN environment variable.

Examples:
  ENVS3_TOKEN=xxx envs3 run --restart="systemctl restart php-fpm"
  ENVS3_TOKEN=xxx envs3 run --user=www-data --interval=60
  ENVS3_TOKEN=xxx envs3 run --restart="pm2 restart all" --interval=10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		token := os.Getenv("ENVS3_TOKEN")
		if token == "" {
			return fmt.Errorf("ENVS3_TOKEN environment variable required for daemon mode")
		}

		payload, err := config.ParseServiceToken(token)
		if err != nil {
			return err
		}

		// Check token expiry
		if payload.ExpiresAt != nil {
			if exp, err := time.Parse(time.RFC3339, *payload.ExpiresAt); err == nil {
				if time.Now().UTC().After(exp) {
					return fmt.Errorf("token expired at %s", *payload.ExpiresAt)
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
			return fmt.Errorf("invalid token keypair: %w", err)
		}
		pub, err := crypto.DecodeKey(payload.PublicKey)
		if err != nil {
			return fmt.Errorf("invalid token keypair: %w", err)
		}

		env := payload.Env
		fp := crypto.Fingerprint(pub)

		// Initial pull
		lastETag, err := doPull(eng, env, priv, pub, "", fp, payload.Name)
		if err != nil {
			return fmt.Errorf("initial pull: %w", err)
		}

		fmt.Printf("envs3 run: watching %s/%s (every %ds)\n", payload.Project, env, runInterval)

		// Watch loop
		ticker := time.NewTicker(time.Duration(runInterval) * time.Second)
		defer ticker.Stop()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		for {
			select {
			case <-ticker.C:
				newETag, err := doPull(eng, env, priv, pub, lastETag, fp, payload.Name)
				if err != nil {
					fmt.Fprintf(os.Stderr, "envs3 run: pull error: %v\n", err)
					continue
				}
				if newETag != lastETag {
					lastETag = newETag
					if runRestart != "" {
						fmt.Printf("envs3 run: executing restart: %s\n", runRestart)
						cmd := exec.Command("sh", "-c", runRestart)
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						if err := cmd.Run(); err != nil {
							fmt.Fprintf(os.Stderr, "envs3 run: restart error: %v\n", err)
						}
					}
				}

			case sig := <-sigCh:
				fmt.Printf("\nenvs3 run: received %s, shutting down\n", sig)
				return nil
			}
		}
	},
}

// doPull fetches secrets and writes .env. Returns the new ETag.
func doPull(eng *engine.Engine, env string, priv, pub [32]byte, cachedETag, fingerprint, actor string) (string, error) {
	result, err := eng.Pull(ctx(), env, priv, pub, cachedETag)
	if err != nil {
		return cachedETag, err
	}

	if result.UpToDate {
		return cachedETag, nil
	}

	// Write .env
	var buf bytes.Buffer
	format.WriteDotEnv(&buf, result.Secrets, result.Environment, result.Version, result.UpdatedAt)

	fileMode := os.FileMode(0644)
	if runUser != "" {
		fileMode = 0600
	}

	if err := os.WriteFile(runOutput, buf.Bytes(), fileMode); err != nil {
		return cachedETag, fmt.Errorf("write %s: %w", runOutput, err)
	}

	if runUser != "" {
		chownFile(runOutput, runUser)
	}

	fmt.Printf("envs3 run: updated %s v%d (%d keys)\n", env, result.Version, len(result.Secrets))

	// Audit
	eng.AuditPull(ctx(), env, actor, fingerprint, result.Version)

	return result.ETag, nil
}
