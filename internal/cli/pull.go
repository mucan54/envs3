package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"sort"
	"strconv"
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
	pullOutput   string
	pullForce    bool
	pullJSONMeta bool
	pullNoHook   bool
	pullToken    string
	pullUser     string
)

func init() {
	pullCmd.Flags().StringVar(&pullOutput, "output", ".env", "Write to custom path")
	pullCmd.Flags().BoolVar(&pullForce, "force", false, "Skip ETag check")
	pullCmd.Flags().BoolVar(&pullJSONMeta, "json-meta", false, "Write metadata to stdout as JSON")
	pullCmd.Flags().BoolVar(&pullNoHook, "no-hook", false, "Skip post_pull hook")
	pullCmd.Flags().StringVar(&pullToken, "token", "", "Service token (alternative to ENVS3_TOKEN env var)")
	pullCmd.Flags().StringVar(&pullUser, "user", "", "Set file owner (e.g., www-data)")
	rootCmd.AddCommand(pullCmd)
}

var pullCmd = &cobra.Command{
	Use:   "pull [environment]",
	Short: "Fetch and decrypt secrets, writing to .env",
	Long: `Fetch and decrypt secrets from S3, writing to .env file.

In development, uses local config (.env.envs3 + private key):
  envs3 pull

In production, uses a service token (no config files needed):
  envs3 pull --token=$TOKEN
  envs3 pull --token=$TOKEN --user=www-data
  ENVS3_TOKEN=xxx envs3 pull --user=www-data`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve token: --token flag > ENVS3_TOKEN env var
		token := pullToken
		if token == "" {
			token = os.Getenv("ENVS3_TOKEN")
		}

		if token != "" {
			return pullWithToken(token, args)
		}
		return pullWithConfig(args)
	},
}

// pullWithToken pulls using a service token (production mode — no config files needed)
func pullWithToken(token string, args []string) error {
	payload, err := config.ParseServiceToken(token)
	if err != nil {
		return err
	}

	// Check token expiry
	if payload.ExpiresAt != nil {
		if exp, err := time.Parse(time.RFC3339, *payload.ExpiresAt); err == nil {
			if time.Now().UTC().After(exp) {
				return fmt.Errorf("token expired at %s. Request a new token from your admin", *payload.ExpiresAt)
			}
		}
	}

	// Build store from token's embedded S3 credentials
	store := storage.NewS3Store(storage.S3Config{
		Endpoint:        payload.Storage.Endpoint,
		Region:          payload.Storage.Region,
		Bucket:          payload.Storage.Bucket,
		AccessKeyID:     payload.Storage.AccessKeyID,
		SecretAccessKey: payload.Storage.SecretAccessKey,
	})

	eng := engine.NewEngine(store, payload.Project)

	// Decode token's keypair
	priv, err := crypto.DecodeKey(payload.PrivateKey)
	if err != nil {
		return fmt.Errorf("invalid token keypair: %w", err)
	}
	pub, err := crypto.DecodeKey(payload.PublicKey)
	if err != nil {
		return fmt.Errorf("invalid token keypair: %w", err)
	}

	env := payload.Env
	if len(args) > 0 {
		env = args[0]
	}

	result, err := eng.Pull(ctx(), env, priv, pub, "")
	if err != nil {
		return err
	}

	// Write .env
	var buf bytes.Buffer
	format.WriteDotEnv(&buf, result.Secrets, result.Environment, result.Version, result.UpdatedAt)

	fileMode := os.FileMode(0644)
	if pullUser != "" {
		fileMode = 0600
	}

	if err := os.WriteFile(pullOutput, buf.Bytes(), fileMode); err != nil {
		return fmt.Errorf("write %s: %w", pullOutput, err)
	}

	// Set file ownership if --user specified
	if pullUser != "" {
		if err := chownFile(pullOutput, pullUser); err != nil {
			return fmt.Errorf("chown %s: %w", pullOutput, err)
		}
	}

	fmt.Printf("✓ Pulled %s v%d (%d keys)\n", env, result.Version, len(result.Secrets))

	// Audit
	eng.AuditPull(ctx(), env, payload.Name, crypto.Fingerprint(pub), result.Version)

	return nil
}

// pullWithConfig pulls using local config files (development mode)
func pullWithConfig(args []string) error {
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

	fileMode := os.FileMode(0644)
	if pullUser != "" {
		fileMode = 0600
	}

	if err := os.WriteFile(pullOutput, buf.Bytes(), fileMode); err != nil {
		return fmt.Errorf("write %s: %w", pullOutput, err)
	}

	if pullUser != "" {
		if err := chownFile(pullOutput, pullUser); err != nil {
			return fmt.Errorf("chown %s: %w", pullOutput, err)
		}
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

	// Expiry & rotation warnings
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
		cmd := exec.Command("sh", "-c", cfg.HookPostPull)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}

	// Audit
	fp := crypto.Fingerprint(pub)
	eng.AuditPull(ctx(), env, "", fp, result.Version)

	return nil
}

// chownFile changes file ownership to the specified user.
func chownFile(path, username string) error {
	u, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("user '%s' not found: %w", username, err)
	}
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)
	return syscall.Chown(path, uid, gid)
}
