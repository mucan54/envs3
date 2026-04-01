package cli

import (
	"encoding/json"
	"fmt"

	"github.com/mucan54/envs3/internal/config"
	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(connectCmd)
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to an existing project",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("envs3 — Connect to Existing Project")
		fmt.Println()

		endpoint := prompt("S3 endpoint URL: ")
		region := prompt("Region (default: auto): ")
		if region == "" {
			region = "auto"
		}
		accessKeyID := prompt("Access Key ID: ")
		secretKey := prompt("Secret Access Key: ")
		bucket := prompt("Bucket name: ")

		store := storage.NewS3Store(storage.S3Config{
			Endpoint:        endpoint,
			Region:          region,
			Bucket:          bucket,
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretKey,
		})

		// Discover projects
		keys, err := store.List(ctx(), "")
		if err != nil {
			return fmt.Errorf("cannot connect to storage: %w", err)
		}

		projects := make(map[string]bool)
		for _, k := range keys {
			for i := 0; i < len(k); i++ {
				if k[i] == '/' {
					projects[k[:i]] = true
					break
				}
			}
		}

		if len(projects) == 0 {
			fmt.Println("No projects found in bucket")
			return nil
		}

		fmt.Println("\nProjects found:")
		for p := range projects {
			fmt.Printf("  - %s\n", p)
		}

		project := prompt("\nProject name: ")

		var projectFile format.ProjectFile
		data, _, err := store.Get(ctx(), storage.ProjectPath(project))
		if err != nil {
			return fmt.Errorf("project '%s' not found: %w", project, err)
		}
		if err := json.Unmarshal(data, &projectFile); err != nil {
			return fmt.Errorf("invalid project: %w", err)
		}

		fmt.Printf("\nProject: %s (%d environments)\n", projectFile.Name, len(projectFile.Environments))
		for _, env := range projectFile.Environments {
			fmt.Printf("  - %s\n", env)
		}

		defaultEnv := "local"
		if len(projectFile.Environments) > 0 {
			defaultEnv = projectFile.Environments[0]
		}

		// Configuration mode
		fmt.Println()
		fmt.Println("Configuration mode:")
		fmt.Println("  1. Hybrid — .envs3.json (commit) + .env.envs3 (secrets only) [default]")
		fmt.Println("  2. Full .env — everything in .env.envs3 (nothing committed)")
		fmt.Println("  3. Full JSON — everything in .envs3.json")
		modeChoice := prompt("Select (1-3) [1]: ")
		if modeChoice == "" {
			modeChoice = "1"
		}

		switch modeChoice {
		case "2": // Full .env
			envCfg := &format.Envs3Config{
				Project:         project,
				Endpoint:        endpoint,
				Bucket:          bucket,
				Region:          region,
				DefaultEnv:      defaultEnv,
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretKey,
			}
			if err := format.SaveEnvs3File(".", envCfg, false); err != nil {
				return err
			}
			fmt.Println("\n✓ .env.envs3 created (do NOT commit)")

		case "3": // Full JSON
			cfg := &format.ProjectConfig{
				SchemaVersion: 1,
				Project:       project,
				Storage: format.StorageConfig{
					Type:                "s3",
					Endpoint:            endpoint,
					Bucket:              bucket,
					Region:              region,
					ReadAccessKeyID:     accessKeyID,
					ReadSecretAccessKey: secretKey,
				},
				Defaults: format.DefaultsConfig{
					Environment: defaultEnv,
				},
			}
			if err := config.SaveProjectConfig(".", cfg); err != nil {
				return err
			}
			fmt.Println("\n✓ .envs3.json created")

		default: // Hybrid
			cfg := &format.ProjectConfig{
				SchemaVersion: 1,
				Project:       project,
				Storage: format.StorageConfig{
					Type:   "s3",
					Bucket: bucket,
					Region: region,
				},
				Defaults: format.DefaultsConfig{
					Environment: defaultEnv,
				},
			}
			if err := config.SaveProjectConfig(".", cfg); err != nil {
				return err
			}
			fmt.Println("\n✓ .envs3.json created (commit this file)")

			envCfg := &format.Envs3Config{
				Endpoint:        endpoint,
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretKey,
			}
			if err := format.SaveEnvs3File(".", envCfg, true); err != nil {
				return err
			}
			fmt.Println("✓ .env.envs3 created (do NOT commit)")
		}

		fmt.Println("\nNext steps:")
		fmt.Println("  1. Run 'envs3 auth setup' to generate your keypair")
		fmt.Println("  2. Send your public key to the admin")
		fmt.Println("  3. Once added, run 'envs3 pull' to sync")

		return nil
	},
}
