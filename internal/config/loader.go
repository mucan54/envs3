package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mucan54/envs3/internal/format"
)

const configFileName = ".envs3.json"

// FindProjectConfig walks up from startDir looking for .envs3.json.
func FindProjectConfig(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		path := filepath.Join(dir, configFileName)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not an envs3 project. Run 'envs3 init' or 'envs3 connect'")
}

// LoadProjectConfig reads and parses .envs3.json.
func LoadProjectConfig(path string) (*format.ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
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

	// Check permissions
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
