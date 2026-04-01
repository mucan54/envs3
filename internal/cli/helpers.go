package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mucan54/envs3/internal/config"
	"github.com/mucan54/envs3/internal/crypto"
	"github.com/mucan54/envs3/internal/engine"
	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
)

// loadContext loads project config, credentials, and creates an engine.
func loadContext() (*engine.Engine, *format.ProjectConfig, [32]byte, [32]byte, error) {
	cfgPath, err := config.FindProjectConfig(getProjectDir())
	if err != nil {
		return nil, nil, [32]byte{}, [32]byte{}, err
	}

	cfg, err := config.LoadProjectConfig(cfgPath)
	if err != nil {
		return nil, nil, [32]byte{}, [32]byte{}, err
	}

	priv, err := config.LoadPrivateKey(cfg.Project)
	if err != nil {
		return nil, nil, [32]byte{}, [32]byte{}, err
	}

	pub, err := crypto.PublicKeyFromPrivate(priv)
	if err != nil {
		return nil, nil, [32]byte{}, [32]byte{}, err
	}

	// Determine S3 credentials (write creds if available, else read-only)
	accessKey := cfg.Storage.ReadAccessKeyID
	secretKey := cfg.Storage.ReadSecretAccessKey

	adminCreds, _ := config.LoadAdminCredentials(cfg.Project)
	if adminCreds != nil && adminCreds.WriteAccessKeyID != "" {
		accessKey = adminCreds.WriteAccessKeyID
		secretKey = adminCreds.WriteSecretAccessKey
	}

	// Check for env var overrides
	if v := os.Getenv("ENVS3_WRITE_KEY_ID"); v != "" {
		accessKey = v
	}
	if v := os.Getenv("ENVS3_WRITE_SECRET_KEY"); v != "" {
		secretKey = v
	}

	store := storage.NewS3Store(storage.S3Config{
		Endpoint:        cfg.Storage.Endpoint,
		Region:          cfg.Storage.Region,
		Bucket:          cfg.Storage.Bucket,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	})

	eng := engine.NewEngine(store, cfg.Project)
	return eng, cfg, priv, pub, nil
}

// loadWriteContext loads context and verifies write credentials are available.
func loadWriteContext() (*engine.Engine, *format.ProjectConfig, [32]byte, [32]byte, error) {
	eng, cfg, priv, pub, err := loadContext()
	if err != nil {
		return nil, nil, [32]byte{}, [32]byte{}, err
	}

	adminCreds, _ := config.LoadAdminCredentials(cfg.Project)
	hasWriteKey := (adminCreds != nil && adminCreds.WriteAccessKeyID != "") ||
		os.Getenv("ENVS3_WRITE_KEY_ID") != ""

	if !hasWriteKey {
		return nil, nil, [32]byte{}, [32]byte{}, fmt.Errorf("read-only access. This operation requires a read-write S3 key. Contact your admin")
	}

	return eng, cfg, priv, pub, nil
}

func resolveEnv(cfg *format.ProjectConfig, envArg string) string {
	if envArg != "" {
		return envArg
	}
	state, err := config.LoadLocalState(cfg.Project)
	if err == nil && state.ActiveEnvironment != "" {
		return state.ActiveEnvironment
	}
	return cfg.Defaults.Environment
}

func ctx() context.Context {
	return context.Background()
}

func prompt(msg string) string {
	fmt.Print(msg)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func confirm(msg string) bool {
	answer := prompt(msg + " (y/N) ")
	return strings.ToLower(answer) == "y"
}

func getUserEmail() string {
	if email := os.Getenv("EMAIL"); email != "" {
		return email
	}
	// Try git config
	// Simple fallback
	return prompt("Email: ")
}
