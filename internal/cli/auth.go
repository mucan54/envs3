package cli

import (
	"fmt"

	"github.com/mucan54/envs3/internal/config"
	"github.com/mucan54/envs3/internal/crypto"
	"github.com/spf13/cobra"
)

func init() {
	authCmd.AddCommand(authSetupCmd)
	rootCmd.AddCommand(authCmd)
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication management",
}

var authSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Generate a new keypair for the current project",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine project name
		var project string
		cfgPath, err := config.FindProjectConfig(getProjectDir())
		if err == nil {
			cfg, err := config.LoadProjectConfig(cfgPath)
			if err == nil {
				project = cfg.Project
			}
		}

		if project == "" {
			project = prompt("Project name: ")
		}

		// Generate keypair
		pub, priv, err := crypto.GenerateKeypair()
		if err != nil {
			return fmt.Errorf("generate keypair: %w", err)
		}

		if err := config.SavePrivateKey(project, priv); err != nil {
			return fmt.Errorf("save private key: %w", err)
		}

		fp := crypto.Fingerprint(pub)
		fmt.Printf("✓ Keypair generated\n")
		fmt.Printf("  Private key: %s\n", config.PrivateKeyPath(project))
		fmt.Printf("  Public key:  %s\n", crypto.EncodeKey(pub))
		fmt.Printf("  Fingerprint: %s\n", fp)
		fmt.Println()
		fmt.Println("Share your public key with the admin:")
		fmt.Printf("  envs3 pubkey --output my.pub\n")

		return nil
	},
}
