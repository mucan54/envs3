package config

import (
	"os"
	"path/filepath"
)

// HomeDir returns the envs3 config directory: ~/.envs3/
func HomeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".envs3")
}

// KeysDir returns ~/.envs3/keys/
func KeysDir() string {
	return filepath.Join(HomeDir(), "keys")
}

// StateDir returns ~/.envs3/state/
func StateDir() string {
	return filepath.Join(HomeDir(), "state")
}

// CredentialsDir returns ~/.envs3/credentials/
func CredentialsDir() string {
	return filepath.Join(HomeDir(), "credentials")
}

// PrivateKeyPath returns the path to a project's private key.
func PrivateKeyPath(project string) string {
	return filepath.Join(KeysDir(), project+".key")
}

// StatePath returns the path to a project's local state file.
func StatePath(project string) string {
	return filepath.Join(StateDir(), project+".json")
}

// CredentialsPath returns the path to a project's admin credentials.
func CredentialsPath(project string) string {
	return filepath.Join(CredentialsDir(), project+".json")
}

// EnsureDirs creates all required directories with 0700 permissions.
func EnsureDirs() error {
	dirs := []string{HomeDir(), KeysDir(), StateDir(), CredentialsDir()}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	return nil
}
