package format

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// .env.envs3 stores S3 credentials (always gitignored).
// In hybrid mode: only endpoint + keys.
// In full-env mode: all project config + credentials.

// Environment variable keys
const (
	EnvProject              = "ENVS3_PROJECT"
	EnvEndpoint             = "ENVS3_ENDPOINT"
	EnvBucket               = "ENVS3_BUCKET"
	EnvRegion               = "ENVS3_REGION"
	EnvDefaultEnv           = "ENVS3_DEFAULT_ENV"
	EnvAccessKeyID          = "ENVS3_ACCESS_KEY_ID"
	EnvSecretAccessKey      = "ENVS3_SECRET_ACCESS_KEY"
	EnvWriteAccessKeyID     = "ENVS3_WRITE_ACCESS_KEY_ID"
	EnvWriteSecretAccessKey = "ENVS3_WRITE_SECRET_ACCESS_KEY"
	EnvHookPostPull         = "ENVS3_HOOK_POST_PULL"
)

// Envs3Config holds configuration loaded from .env.envs3.
// In hybrid mode, only credential fields are populated.
// In full-env mode, all fields are populated.
type Envs3Config struct {
	// Project config (used in full-env mode, empty in hybrid)
	Project      string
	Bucket       string
	Region       string
	DefaultEnv   string
	HookPostPull string

	// S3 credentials (always used)
	Endpoint             string
	AccessKeyID          string
	SecretAccessKey      string
	WriteAccessKeyID     string
	WriteSecretAccessKey string
}

// ParseEnvs3File reads a .env.envs3 file.
func ParseEnvs3File(r io.Reader) (*Envs3Config, error) {
	kv, err := ParseDotEnv(r)
	if err != nil {
		return nil, err
	}

	return &Envs3Config{
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
	}, nil
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
// If credentialsOnly is true (hybrid mode), only writes endpoint + S3 keys.
// If credentialsOnly is false (full-env mode), writes all fields.
func WriteEnvs3File(w io.Writer, cfg *Envs3Config, credentialsOnly bool) error {
	var lines []string

	if credentialsOnly {
		lines = []string{
			"# envs3 storage credentials",
			"# This file is gitignored. Share it with your team via a secure channel.",
			"",
			fmt.Sprintf("%s=%s", EnvEndpoint, cfg.Endpoint),
			fmt.Sprintf("%s=%s", EnvAccessKeyID, cfg.AccessKeyID),
			fmt.Sprintf("%s=%s", EnvSecretAccessKey, cfg.SecretAccessKey),
		}
	} else {
		lines = []string{
			"# envs3 configuration (full mode)",
			"# This file is gitignored. Share it with your team via a secure channel.",
			"",
			"# Project",
			fmt.Sprintf("%s=%s", EnvProject, cfg.Project),
			fmt.Sprintf("%s=%s", EnvBucket, cfg.Bucket),
		}
		if cfg.Region != "" {
			lines = append(lines, fmt.Sprintf("%s=%s", EnvRegion, cfg.Region))
		}
		if cfg.DefaultEnv != "" {
			lines = append(lines, fmt.Sprintf("%s=%s", EnvDefaultEnv, cfg.DefaultEnv))
		}
		if cfg.HookPostPull != "" {
			lines = append(lines, fmt.Sprintf("%s=%s", EnvHookPostPull, cfg.HookPostPull))
		}
		lines = append(lines,
			"",
			"# Storage credentials",
			fmt.Sprintf("%s=%s", EnvEndpoint, cfg.Endpoint),
			fmt.Sprintf("%s=%s", EnvAccessKeyID, cfg.AccessKeyID),
			fmt.Sprintf("%s=%s", EnvSecretAccessKey, cfg.SecretAccessKey),
		)
	}

	// Write credentials (admin section)
	lines = append(lines, "")
	if cfg.WriteAccessKeyID != "" {
		lines = append(lines,
			"# Write credentials (admin only)",
			fmt.Sprintf("%s=%s", EnvWriteAccessKeyID, cfg.WriteAccessKeyID),
			fmt.Sprintf("%s=%s", EnvWriteSecretAccessKey, cfg.WriteSecretAccessKey),
		)
	} else {
		lines = append(lines,
			"# Write credentials (admin only — uncomment if you have read-write access)",
			fmt.Sprintf("# %s=", EnvWriteAccessKeyID),
			fmt.Sprintf("# %s=", EnvWriteSecretAccessKey),
		)
	}

	lines = append(lines, "")
	_, err := io.WriteString(w, strings.Join(lines, "\n"))
	return err
}

// SaveEnvs3File writes .env.envs3 to the given directory with mode 0600.
func SaveEnvs3File(dir string, cfg *Envs3Config, credentialsOnly bool) error {
	path := dir + "/.env.envs3"
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteEnvs3File(f, cfg, credentialsOnly)
}

// ApplyEnvOverrides applies process environment variable overrides (highest priority).
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
