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

// loadContext loads the full config from .env.envs3 (+ optional .envs3.json)
// and creates an engine.
func loadContext() (*engine.Engine, *format.Envs3Config, [32]byte, [32]byte, error) {
	cfg, _, err := config.LoadFullConfig(getProjectDir())
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

	// Use write credentials if available, otherwise read-only
	accessKey := cfg.AccessKeyID
	secretKey := cfg.SecretAccessKey
	if cfg.HasWriteCredentials() {
		accessKey = cfg.WriteAccessKeyID
		secretKey = cfg.WriteSecretAccessKey
	}

	store := storage.NewS3Store(storage.S3Config{
		Endpoint:        cfg.Endpoint,
		Region:          cfg.Region,
		Bucket:          cfg.Bucket,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	})

	eng := engine.NewEngine(store, cfg.Project)
	return eng, cfg, priv, pub, nil
}

// loadWriteContext loads context and verifies write credentials are available.
func loadWriteContext() (*engine.Engine, *format.Envs3Config, [32]byte, [32]byte, error) {
	eng, cfg, priv, pub, err := loadContext()
	if err != nil {
		return nil, nil, [32]byte{}, [32]byte{}, err
	}

	if !cfg.HasWriteCredentials() {
		return nil, nil, [32]byte{}, [32]byte{}, fmt.Errorf("read-only access. This operation requires read-write S3 credentials.\n\n  Add ENVS3_WRITE_ACCESS_KEY_ID and ENVS3_WRITE_SECRET_ACCESS_KEY\n  to your .env.envs3 file, or contact your admin")
	}

	return eng, cfg, priv, pub, nil
}

func resolveEnv(cfg *format.Envs3Config, envArg string) string {
	if envArg != "" {
		return envArg
	}
	state, err := config.LoadLocalState(cfg.Project)
	if err == nil && state.ActiveEnvironment != "" {
		return state.ActiveEnvironment
	}
	return cfg.DefaultEnv
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
	return prompt("Email: ")
}
