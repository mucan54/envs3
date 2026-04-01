package cli

import (
	"fmt"
	"os"

	"github.com/mucan54/envs3/internal/config"
	"github.com/mucan54/envs3/internal/format"
	"github.com/spf13/cobra"
)

var jsonKeepS3 bool

func init() {
	jsonCmd.Flags().BoolVar(&jsonKeepS3, "keep-s3", false, "Keep S3 credentials in .env.envs3, move only project config to JSON")
	rootCmd.AddCommand(jsonCmd)
}

var jsonCmd = &cobra.Command{
	Use:   "json",
	Short: "Convert .env.envs3 config to .envs3.json",
	Long: `Convert configuration from .env.envs3 to .envs3.json format.

By default, moves everything to .envs3.json and removes .env.envs3.
With --keep-s3, moves only project config (name, bucket, region, defaults,
hooks) to .envs3.json and keeps S3 credentials in .env.envs3.

Examples:
  envs3 json            # Everything → .envs3.json (removes .env.envs3)
  envs3 json --keep-s3  # Project config → .envs3.json, credentials stay in .env.envs3`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Find and load current config
		cfg, projectDir, err := config.LoadFullConfig(getProjectDir())
		if err != nil {
			return err
		}

		if jsonKeepS3 {
			// --keep-s3: project config → .envs3.json, credentials stay in .env.envs3
			jsonCfg := &format.ProjectConfig{
				SchemaVersion: 1,
				Project:       cfg.Project,
				Storage: format.StorageConfig{
					Type:   "s3",
					Bucket: cfg.Bucket,
					Region: cfg.Region,
				},
				Defaults: format.DefaultsConfig{
					Environment: cfg.DefaultEnv,
				},
			}
			if cfg.HookPostPull != "" {
				jsonCfg.Hooks = &format.HooksConfig{PostPull: cfg.HookPostPull}
			}

			if err := config.SaveProjectConfig(projectDir, jsonCfg); err != nil {
				return fmt.Errorf("write .envs3.json: %w", err)
			}

			// Rewrite .env.envs3 with credentials only
			envCfg := &format.Envs3Config{
				Endpoint:             cfg.Endpoint,
				AccessKeyID:          cfg.AccessKeyID,
				SecretAccessKey:      cfg.SecretAccessKey,
				WriteAccessKeyID:     cfg.WriteAccessKeyID,
				WriteSecretAccessKey: cfg.WriteSecretAccessKey,
			}
			if err := format.SaveEnvs3File(projectDir, envCfg, true); err != nil {
				return fmt.Errorf("write .env.envs3: %w", err)
			}

			fmt.Println("✓ .envs3.json created (project config — commit this file)")
			fmt.Println("✓ .env.envs3 updated (S3 credentials only)")

		} else {
			// Default: everything → .envs3.json, remove .env.envs3
			jsonCfg := &format.ProjectConfig{
				SchemaVersion: 1,
				Project:       cfg.Project,
				Storage: format.StorageConfig{
					Type:                "s3",
					Endpoint:            cfg.Endpoint,
					Bucket:              cfg.Bucket,
					Region:              cfg.Region,
					ReadAccessKeyID:     cfg.AccessKeyID,
					ReadSecretAccessKey: cfg.SecretAccessKey,
				},
				Defaults: format.DefaultsConfig{
					Environment: cfg.DefaultEnv,
				},
			}
			if cfg.HookPostPull != "" {
				jsonCfg.Hooks = &format.HooksConfig{PostPull: cfg.HookPostPull}
			}

			if err := config.SaveProjectConfig(projectDir, jsonCfg); err != nil {
				return fmt.Errorf("write .envs3.json: %w", err)
			}

			// Remove .env.envs3
			envPath := projectDir + "/.env.envs3"
			if _, err := os.Stat(envPath); err == nil {
				if err := os.Remove(envPath); err != nil {
					return fmt.Errorf("remove .env.envs3: %w", err)
				}
			}

			fmt.Println("✓ .envs3.json created (contains all config including credentials)")
			fmt.Println("✓ .env.envs3 removed")
			fmt.Println()
			fmt.Println("Note: .envs3.json now contains S3 credentials.")
			fmt.Println("      Add it to .gitignore if you don't want to commit credentials.")
		}

		return nil
	},
}
