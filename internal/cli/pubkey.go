package cli

import (
	"fmt"
	"os"

	"github.com/mucan54/envs3/internal/config"
	"github.com/mucan54/envs3/internal/crypto"
	"github.com/spf13/cobra"
)

var (
	pubkeyOutput string
)

func init() {
	pubkeyCmd.Flags().StringVar(&pubkeyOutput, "output", "", "Write to file")
	rootCmd.AddCommand(pubkeyCmd)
}

var pubkeyCmd = &cobra.Command{
	Use:   "pubkey",
	Short: "Display the current user's public key",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := config.FindProjectConfig(getProjectDir())
		if err != nil {
			return err
		}
		cfg, err := config.LoadProjectConfig(cfgPath)
		if err != nil {
			return err
		}

		priv, err := config.LoadPrivateKey(cfg.Project)
		if err != nil {
			return err
		}

		pub, err := crypto.PublicKeyFromPrivate(priv)
		if err != nil {
			return err
		}

		email := getUserEmail()
		fp := crypto.Fingerprint(pub)

		line := fmt.Sprintf("envs3-pubkey-v1 %s %s", crypto.EncodeKey(pub), email)

		if pubkeyOutput != "" {
			if err := os.WriteFile(pubkeyOutput, []byte(line+"\n"), 0644); err != nil {
				return err
			}
			fmt.Printf("✓ Public key written to %s\n", pubkeyOutput)
		} else {
			fmt.Println(line)
		}

		fmt.Printf("Fingerprint: %s\n", fp)
		return nil
	},
}
