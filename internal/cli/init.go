package cli

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/mucan54/envs3/internal/config"
	"github.com/mucan54/envs3/internal/crypto"
	"github.com/mucan54/envs3/internal/engine"
	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
	"github.com/spf13/cobra"
)

var projectNameRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new envs3 project",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("envs3 — Project Initialization")
		fmt.Println()

		// Storage backend
		fmt.Println("Storage backend:")
		fmt.Println("  1. Cloudflare R2")
		fmt.Println("  2. AWS S3")
		fmt.Println("  3. MinIO")
		fmt.Println("  4. Custom S3-compatible")
		choice := prompt("Select (1-4): ")

		var endpoint, region string
		switch choice {
		case "1":
			endpoint = prompt("R2 endpoint URL: ")
			region = "auto"
		case "2":
			region = prompt("AWS region (e.g., us-east-1): ")
			endpoint = fmt.Sprintf("https://s3.%s.amazonaws.com", region)
		case "3":
			endpoint = prompt("MinIO endpoint URL: ")
			region = "us-east-1"
		default:
			endpoint = prompt("S3 endpoint URL: ")
			region = prompt("Region (default: auto): ")
			if region == "" {
				region = "auto"
			}
		}

		// Read-only credentials
		fmt.Println()
		fmt.Println("Read-only S3 credentials (for team members):")
		readKeyID := prompt("Read-only Access Key ID: ")
		readSecretKey := prompt("Read-only Secret Access Key: ")

		// Write credentials
		fmt.Println()
		fmt.Println("Admin S3 credentials (read-write access):")
		writeKeyID := prompt("Read-write Access Key ID: ")
		writeSecretKey := prompt("Read-write Secret Access Key: ")

		bucket := prompt("Bucket name: ")

		projectName := prompt("Project name: ")
		if !projectNameRegex.MatchString(projectName) {
			return fmt.Errorf("project name must match [a-z0-9-]+")
		}

		email := getUserEmail()

		// Environments
		envsInput := prompt("Environments (comma-separated, default: local,staging,production): ")
		if envsInput == "" {
			envsInput = "local,staging,production"
		}
		envs := strings.Split(envsInput, ",")
		for i := range envs {
			envs[i] = strings.TrimSpace(envs[i])
		}

		// Configuration mode
		fmt.Println()
		fmt.Println("Configuration mode:")
		fmt.Println("  1. Hybrid — .envs3.json (commit) + .env.envs3 (secrets only) [default]")
		fmt.Println("  2. Full .env — everything in .env.envs3 (nothing committed)")
		fmt.Println("  3. Full JSON — everything in .envs3.json (you decide what to commit)")
		modeChoice := prompt("Select (1-3) [1]: ")
		if modeChoice == "" {
			modeChoice = "1"
		}

		// Generate keypair
		pub, priv, err := crypto.GenerateKeypair()
		if err != nil {
			return fmt.Errorf("generate keypair: %w", err)
		}

		if err := config.SavePrivateKey(projectName, priv); err != nil {
			return fmt.Errorf("save private key: %w", err)
		}
		fmt.Printf("✓ Keypair generated → %s\n", config.PrivateKeyPath(projectName))

		// Check for .env import
		var importEnv string
		var importData map[string]string
		if _, err := os.Stat(".env"); err == nil {
			if confirm("Import existing .env?") {
				importEnv = prompt("Import into environment (default: local): ")
				if importEnv == "" {
					importEnv = "local"
				}
				f, err := os.Open(".env")
				if err != nil {
					return err
				}
				defer f.Close()
				importData, err = format.ParseDotEnv(f)
				if err != nil {
					return fmt.Errorf("parse .env: %w", err)
				}
			}
		}

		// Create S3 store with write credentials
		store := storage.NewS3Store(storage.S3Config{
			Endpoint:        endpoint,
			Region:          region,
			Bucket:          bucket,
			AccessKeyID:     writeKeyID,
			SecretAccessKey: writeSecretKey,
		})

		eng := engine.NewEngine(store, projectName)
		if err := eng.Init(ctx(), engine.InitParams{
			ProjectName:  projectName,
			Email:        email,
			Environments: envs,
			ImportEnv:    importEnv,
			ImportData:   importData,
			PublicKey:    pub,
		}); err != nil {
			return fmt.Errorf("init: %w", err)
		}

		fmt.Printf("✓ Project created → s3://%s/%s/\n", bucket, projectName)
		if importData != nil {
			fmt.Printf("✓ Environment '%s' created (%d keys imported)\n", importEnv, len(importData))
		}

		// Write config files based on mode
		switch modeChoice {
		case "2": // Full .env mode
			envCfg := &format.Envs3Config{
				Project:              projectName,
				Endpoint:             endpoint,
				Bucket:               bucket,
				Region:               region,
				DefaultEnv:           envs[0],
				AccessKeyID:          readKeyID,
				SecretAccessKey:      readSecretKey,
				WriteAccessKeyID:     writeKeyID,
				WriteSecretAccessKey: writeSecretKey,
			}
			if err := format.SaveEnvs3File(".", envCfg, false); err != nil {
				return fmt.Errorf("save .env.envs3: %w", err)
			}
			fmt.Println("✓ .env.envs3 created (do NOT commit — share securely with team)")

		case "3": // Full JSON mode
			cfg := &format.ProjectConfig{
				SchemaVersion: 1,
				Project:       projectName,
				Storage: format.StorageConfig{
					Type:                "s3",
					Endpoint:            endpoint,
					Bucket:              bucket,
					Region:              region,
					ReadAccessKeyID:     readKeyID,
					ReadSecretAccessKey: readSecretKey,
				},
				Defaults: format.DefaultsConfig{
					Environment: envs[0],
				},
			}
			if err := config.SaveProjectConfig(".", cfg); err != nil {
				return fmt.Errorf("save .envs3.json: %w", err)
			}
			fmt.Println("✓ .envs3.json created (contains credentials — add to .gitignore if needed)")

		default: // Hybrid mode (1)
			// .envs3.json — project config, no credentials
			cfg := &format.ProjectConfig{
				SchemaVersion: 1,
				Project:       projectName,
				Storage: format.StorageConfig{
					Type:   "s3",
					Bucket: bucket,
					Region: region,
				},
				Defaults: format.DefaultsConfig{
					Environment: envs[0],
				},
			}
			if err := config.SaveProjectConfig(".", cfg); err != nil {
				return fmt.Errorf("save .envs3.json: %w", err)
			}
			fmt.Println("✓ .envs3.json created (commit this file)")

			// .env.envs3 — credentials only
			envCfg := &format.Envs3Config{
				Endpoint:             endpoint,
				AccessKeyID:          readKeyID,
				SecretAccessKey:      readSecretKey,
				WriteAccessKeyID:     writeKeyID,
				WriteSecretAccessKey: writeSecretKey,
			}
			if err := format.SaveEnvs3File(".", envCfg, true); err != nil {
				return fmt.Errorf("save .env.envs3: %w", err)
			}
			fmt.Println("✓ .env.envs3 created (do NOT commit — share securely with team)")
		}

		// Save initial local state
		state := &format.LocalState{
			Environments:      make(map[string]format.EnvState),
			ActiveEnvironment: envs[0],
		}
		if err := config.SaveLocalState(projectName, state); err != nil {
			return fmt.Errorf("save state: %w", err)
		}

		fmt.Println()
		switch modeChoice {
		case "2":
			fmt.Println("Next steps:")
			fmt.Println("  1. Add .env.envs3 to .gitignore")
			fmt.Println("  2. Share .env.envs3 with your team via a secure channel")
			fmt.Println("  3. Run 'envs3 pull' to sync")
		case "3":
			fmt.Println("Next steps:")
			fmt.Println("  1. Decide whether to commit or gitignore .envs3.json")
			fmt.Println("  2. Run 'envs3 pull' to sync")
		default:
			fmt.Println("Next steps:")
			fmt.Println("  1. Commit .envs3.json to your repository")
			fmt.Println("  2. Share .env.envs3 with your team via a secure channel")
			fmt.Println("  3. Run 'envs3 pull' to sync")
		}

		return nil
	},
}
