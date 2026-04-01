package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	appVersion = "dev"
	verbose    bool
	jsonOutput bool
	projectDir string
)

// SetVersion sets the application version.
func SetVersion(v string) {
	appVersion = v
}

var rootCmd = &cobra.Command{
	Use:   "envs3",
	Short: "Encrypted environment variable management over S3-compatible storage",
	Long: `envs3 manages encrypted environment variables stored on any S3-compatible
storage (Cloudflare R2, AWS S3, MinIO, DigitalOcean Spaces).

Secrets are encrypted client-side using envelope encryption — the storage
backend only ever sees ciphertext.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().StringVarP(&projectDir, "project", "p", "", "Override project directory")
	rootCmd.Version = appVersion
}

// Execute runs the root command.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "✗ %s\n", err)
		return err
	}
	return nil
}

func getProjectDir() string {
	if projectDir != "" {
		return projectDir
	}
	dir, _ := os.Getwd()
	return dir
}
