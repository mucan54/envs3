package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mucan54/envs3/internal/format"
)

const configFileName = ".envs3.json"
const envFileName = ".env.envs3"

// FindProjectDir walks up from startDir looking for .env.envs3 or .envs3.json.
// .env.envs3 takes priority.
func FindProjectDir(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		// Check for .env.envs3 first
		if _, err := os.Stat(filepath.Join(dir, envFileName)); err == nil {
			return dir, nil
		}
		// Fall back to .envs3.json
		if _, err := os.Stat(filepath.Join(dir, configFileName)); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not an envs3 project. Run 'envs3 init' or 'envs3 connect'")
}

// FindProjectConfig walks up from startDir looking for .envs3.json.
// Kept for backward compatibility.
func FindProjectConfig(startDir string) (string, error) {
	dir, err := FindProjectDir(startDir)
	if err != nil {
		return "", err
	}
	// Return .envs3.json path even if it doesn't exist (LoadProjectConfig handles that)
	return filepath.Join(dir, configFileName), nil
}

// LoadFullConfig loads the complete envs3 configuration.
// Priority: .env.envs3 (primary) → .envs3.json (optional overlay) → env vars (highest).
// Returns the resolved config and the project directory.
func LoadFullConfig(startDir string) (*format.Envs3Config, string, error) {
	projectDir, err := FindProjectDir(startDir)
	if err != nil {
		return nil, "", err
	}

	cfg := &format.Envs3Config{}

	// 1. Load from .envs3.json if it exists
	jsonPath := filepath.Join(projectDir, configFileName)
	if data, err := os.ReadFile(jsonPath); err == nil {
		var jsonCfg format.ProjectConfig
		if err := json.Unmarshal(data, &jsonCfg); err == nil {
			cfg.Project = jsonCfg.Project
			cfg.Bucket = jsonCfg.Storage.Bucket
			cfg.Region = jsonCfg.Storage.Region
			cfg.DefaultEnv = jsonCfg.Defaults.Environment
			if jsonCfg.Hooks != nil {
				cfg.HookPostPull = jsonCfg.Hooks.PostPull
			}
			// Full-JSON mode: credentials may also be in .envs3.json
			if jsonCfg.Storage.Endpoint != "" {
				cfg.Endpoint = jsonCfg.Storage.Endpoint
			}
			if jsonCfg.Storage.ReadAccessKeyID != "" {
				cfg.AccessKeyID = jsonCfg.Storage.ReadAccessKeyID
				cfg.SecretAccessKey = jsonCfg.Storage.ReadSecretAccessKey
			}
		}
	}

	// 2. Load from .env.envs3 (overrides .envs3.json values)
	envCfg, err := format.LoadEnvs3File(projectDir)
	if err != nil {
		return nil, "", fmt.Errorf("read .env.envs3: %w", err)
	}
	if envCfg != nil {
		if envCfg.Project != "" {
			cfg.Project = envCfg.Project
		}
		if envCfg.Endpoint != "" {
			cfg.Endpoint = envCfg.Endpoint
		}
		if envCfg.Bucket != "" {
			cfg.Bucket = envCfg.Bucket
		}
		if envCfg.Region != "" {
			cfg.Region = envCfg.Region
		}
		if envCfg.DefaultEnv != "" {
			cfg.DefaultEnv = envCfg.DefaultEnv
		}
		if envCfg.AccessKeyID != "" {
			cfg.AccessKeyID = envCfg.AccessKeyID
		}
		if envCfg.SecretAccessKey != "" {
			cfg.SecretAccessKey = envCfg.SecretAccessKey
		}
		if envCfg.WriteAccessKeyID != "" {
			cfg.WriteAccessKeyID = envCfg.WriteAccessKeyID
		}
		if envCfg.WriteSecretAccessKey != "" {
			cfg.WriteSecretAccessKey = envCfg.WriteSecretAccessKey
		}
		if envCfg.HookPostPull != "" {
			cfg.HookPostPull = envCfg.HookPostPull
		}
	}

	// 3. Legacy: check ~/.envs3/credentials/ for write creds
	if !cfg.HasWriteCredentials() && cfg.Project != "" {
		legacyAdmin, _ := LoadAdminCredentials(cfg.Project)
		if legacyAdmin != nil && legacyAdmin.WriteAccessKeyID != "" {
			cfg.WriteAccessKeyID = legacyAdmin.WriteAccessKeyID
			cfg.WriteSecretAccessKey = legacyAdmin.WriteSecretAccessKey
		}
	}

	// 4. Apply environment variable overrides (highest priority)
	cfg.ApplyEnvOverrides()

	// Validate
	if cfg.Project == "" {
		return nil, "", fmt.Errorf("no project configured. Set ENVS3_PROJECT in .env.envs3 or create .envs3.json")
	}
	if !cfg.HasReadCredentials() {
		return nil, "", fmt.Errorf("no storage credentials found. Set ENVS3_ENDPOINT, ENVS3_ACCESS_KEY_ID, ENVS3_SECRET_ACCESS_KEY in .env.envs3 or as environment variables")
	}

	return cfg, projectDir, nil
}

// LoadProjectConfig reads and parses .envs3.json.
func LoadProjectConfig(path string) (*format.ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &format.ProjectConfig{SchemaVersion: 1}, nil
		}
		return nil, err
	}
	var cfg format.ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}

// SaveProjectConfig writes .envs3.json to the given directory.
func SaveProjectConfig(dir string, cfg *format.ProjectConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, configFileName), data, 0644)
}

// LoadLocalState reads the local state file for a project.
func LoadLocalState(project string) (*format.LocalState, error) {
	data, err := os.ReadFile(StatePath(project))
	if err != nil {
		if os.IsNotExist(err) {
			return &format.LocalState{
				Environments: make(map[string]format.EnvState),
			}, nil
		}
		return nil, err
	}
	var state format.LocalState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	if state.Environments == nil {
		state.Environments = make(map[string]format.EnvState)
	}
	return &state, nil
}

// SaveLocalState writes the local state file.
func SaveLocalState(project string, state *format.LocalState) error {
	if err := EnsureDirs(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(StatePath(project), data, 0600)
}

// LoadPrivateKey reads the raw 32-byte private key for a project.
func LoadPrivateKey(project string) ([32]byte, error) {
	var key [32]byte
	path := PrivateKeyPath(project)

	info, err := os.Stat(path)
	if err != nil {
		return key, fmt.Errorf("private key not found at %s. Run 'envs3 auth setup'", path)
	}

	if info.Mode().Perm()&0077 != 0 {
		return key, fmt.Errorf("private key file has insecure permissions (%o). Run: chmod 600 %s", info.Mode().Perm(), path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return key, err
	}
	if len(data) != 32 {
		return key, fmt.Errorf("invalid private key: expected 32 bytes, got %d", len(data))
	}
	copy(key[:], data)
	return key, nil
}

// SavePrivateKey writes the raw 32-byte private key with 0600 permissions.
func SavePrivateKey(project string, key [32]byte) error {
	if err := EnsureDirs(); err != nil {
		return err
	}
	return os.WriteFile(PrivateKeyPath(project), key[:], 0600)
}

// LoadAdminCredentials reads the admin write credentials for a project.
// Deprecated: use .env.envs3 with ENVS3_WRITE_ACCESS_KEY_ID instead.
func LoadAdminCredentials(project string) (*format.AdminCredentials, error) {
	data, err := os.ReadFile(CredentialsPath(project))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var creds format.AdminCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// SaveAdminCredentials writes the admin write credentials with 0600 permissions.
func SaveAdminCredentials(project string, creds *format.AdminCredentials) error {
	if err := EnsureDirs(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(CredentialsPath(project), data, 0600)
}
