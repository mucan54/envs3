package format

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// .env.envs3 is the primary configuration file for envs3.
// It uses the familiar .env format and is gitignored by default.
//
// Example:
//
//   # envs3 project configuration
//   ENVS3_PROJECT=myproject
//   ENVS3_ENDPOINT=https://xxx.r2.cloudflarestorage.com
//   ENVS3_BUCKET=myproject-envs
//   ENVS3_REGION=auto
//   ENVS3_DEFAULT_ENV=local
//   ENVS3_ACCESS_KEY_ID=readonly_abc123
//   ENVS3_SECRET_ACCESS_KEY=readonly_secret_xyz
//   # ENVS3_WRITE_ACCESS_KEY_ID=readwrite_def456
//   # ENVS3_WRITE_SECRET_ACCESS_KEY=readwrite_secret_uvw
//   # ENVS3_HOOK_POST_PULL=php artisan config:clear

// Environment variable keys for .env.envs3
const (
	EnvProject            = "ENVS3_PROJECT"
	EnvEndpoint           = "ENVS3_ENDPOINT"
	EnvBucket             = "ENVS3_BUCKET"
	EnvRegion             = "ENVS3_REGION"
	EnvDefaultEnv         = "ENVS3_DEFAULT_ENV"
	EnvAccessKeyID        = "ENVS3_ACCESS_KEY_ID"
	EnvSecretAccessKey    = "ENVS3_SECRET_ACCESS_KEY"
	EnvWriteAccessKeyID   = "ENVS3_WRITE_ACCESS_KEY_ID"
	EnvWriteSecretAccessKey = "ENVS3_WRITE_SECRET_ACCESS_KEY"
	EnvHookPostPull       = "ENVS3_HOOK_POST_PULL"
)

// Envs3Config holds the full configuration loaded from .env.envs3.
type Envs3Config struct {
	// Project identity
	Project    string
	Bucket     string
	Region     string
	Endpoint   string
	DefaultEnv string

	// Read credentials
	AccessKeyID     string
	SecretAccessKey string

	// Write credentials (admin only)
	WriteAccessKeyID     string
	WriteSecretAccessKey string

	// Hooks
	HookPostPull string
}

// ParseEnvs3File reads a .env.envs3 file and returns the full config.
func ParseEnvs3File(r io.Reader) (*Envs3Config, error) {
	kv, err := ParseDotEnv(r)
	if err != nil {
		return nil, err
	}

	cfg := &Envs3Config{
		Project:              kv[EnvProject],
		Endpoint:             kv[EnvEndpoint],
		Bucket:               kv[EnvBucket],
		Region:               kv[EnvRegion],
		DefaultEnv:           kv[EnvDefaultEnv],
		AccessKeyID:          kv[EnvAccessKeyID],
		SecretAccessKey:      kv[EnvSecretAccessKey],
		WriteAccessKeyID:     kv[EnvWriteAccessKeyID],
		WriteSecretAccessKey: kv[EnvWriteSecretAccessKey],
		HookPostPull:         kv[EnvHookPostPull],
	}

	return cfg, nil
}

// LoadEnvs3File reads .env.envs3 from the given directory.
// Returns nil (no error) if the file doesn't exist.
func LoadEnvs3File(dir string) (*Envs3Config, error) {
	path := dir + "/.env.envs3"
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	return ParseEnvs3File(f)
}

// WriteEnvs3File writes a .env.envs3 file.
func WriteEnvs3File(w io.Writer, cfg *Envs3Config) error {
	lines := []string{
		"# envs3 project configuration",
		"# This file is gitignored. Share it with your team via a secure channel.",
		"",
		"# Project",
		fmt.Sprintf("%s=%s", EnvProject, cfg.Project),
		"",
		"# Storage",
		fmt.Sprintf("%s=%s", EnvEndpoint, cfg.Endpoint),
		fmt.Sprintf("%s=%s", EnvBucket, cfg.Bucket),
	}

	if cfg.Region != "" {
		lines = append(lines, fmt.Sprintf("%s=%s", EnvRegion, cfg.Region))
	}

	lines = append(lines, "")
	if cfg.DefaultEnv != "" {
		lines = append(lines, fmt.Sprintf("%s=%s", EnvDefaultEnv, cfg.DefaultEnv))
		lines = append(lines, "")
	}

	lines = append(lines,
		"# Read credentials (for all team members)",
		fmt.Sprintf("%s=%s", EnvAccessKeyID, cfg.AccessKeyID),
		fmt.Sprintf("%s=%s", EnvSecretAccessKey, cfg.SecretAccessKey),
		"",
		"# Write credentials (admin only — uncomment if you have read-write access)",
	)

	if cfg.WriteAccessKeyID != "" {
		lines = append(lines,
			fmt.Sprintf("%s=%s", EnvWriteAccessKeyID, cfg.WriteAccessKeyID),
			fmt.Sprintf("%s=%s", EnvWriteSecretAccessKey, cfg.WriteSecretAccessKey),
		)
	} else {
		lines = append(lines,
			fmt.Sprintf("# %s=", EnvWriteAccessKeyID),
			fmt.Sprintf("# %s=", EnvWriteSecretAccessKey),
		)
	}

	if cfg.HookPostPull != "" {
		lines = append(lines, "", fmt.Sprintf("%s=%s", EnvHookPostPull, cfg.HookPostPull))
	}

	lines = append(lines, "")
	_, err := io.WriteString(w, strings.Join(lines, "\n"))
	return err
}

// SaveEnvs3File writes .env.envs3 to the given directory with mode 0600.
func SaveEnvs3File(dir string, cfg *Envs3Config) error {
	path := dir + "/.env.envs3"
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteEnvs3File(f, cfg)
}

// ApplyEnvOverrides applies process environment variable overrides.
// Environment variables take highest priority over file values.
func (c *Envs3Config) ApplyEnvOverrides() {
	if v := os.Getenv(EnvProject); v != "" {
		c.Project = v
	}
	if v := os.Getenv(EnvEndpoint); v != "" {
		c.Endpoint = v
	}
	if v := os.Getenv(EnvBucket); v != "" {
		c.Bucket = v
	}
	if v := os.Getenv(EnvRegion); v != "" {
		c.Region = v
	}
	if v := os.Getenv(EnvDefaultEnv); v != "" {
		c.DefaultEnv = v
	}
	if v := os.Getenv(EnvAccessKeyID); v != "" {
		c.AccessKeyID = v
	}
	if v := os.Getenv(EnvSecretAccessKey); v != "" {
		c.SecretAccessKey = v
	}
	if v := os.Getenv(EnvWriteAccessKeyID); v != "" {
		c.WriteAccessKeyID = v
	}
	if v := os.Getenv(EnvWriteSecretAccessKey); v != "" {
		c.WriteSecretAccessKey = v
	}
	if v := os.Getenv(EnvHookPostPull); v != "" {
		c.HookPostPull = v
	}
}

// HasReadCredentials returns true if read credentials are configured.
func (c *Envs3Config) HasReadCredentials() bool {
	return c != nil && c.Endpoint != "" && c.AccessKeyID != "" && c.SecretAccessKey != ""
}

// HasWriteCredentials returns true if write credentials are configured.
func (c *Envs3Config) HasWriteCredentials() bool {
	return c != nil && c.WriteAccessKeyID != "" && c.WriteSecretAccessKey != ""
}

// ToProjectConfig converts to a ProjectConfig (for backward compat with .envs3.json).
func (c *Envs3Config) ToProjectConfig() *ProjectConfig {
	cfg := &ProjectConfig{
		SchemaVersion: 1,
		Project:       c.Project,
		Storage: StorageConfig{
			Type:   "s3",
			Bucket: c.Bucket,
			Region: c.Region,
		},
		Defaults: DefaultsConfig{
			Environment: c.DefaultEnv,
		},
	}
	if c.HookPostPull != "" {
		cfg.Hooks = &HooksConfig{PostPull: c.HookPostPull}
	}
	return cfg
}
