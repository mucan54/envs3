package format

// ProjectConfig represents .envs3.json committed to the repository.
// This file contains NO credentials — only project metadata, defaults, and hooks.
type ProjectConfig struct {
	SchemaVersion int            `json:"schema_version"`
	Project       string         `json:"project"`
	Storage       StorageConfig  `json:"storage"`
	Defaults      DefaultsConfig `json:"defaults"`
	Hooks         *HooksConfig   `json:"hooks,omitempty"`
}

// StorageConfig holds storage configuration.
// In hybrid mode, only type/bucket/region are set (credentials in .env.envs3).
// In full-JSON mode, all fields including endpoint and credentials are set.
type StorageConfig struct {
	Type                string `json:"type"`
	Endpoint            string `json:"endpoint,omitempty"`
	Bucket              string `json:"bucket"`
	Region              string `json:"region,omitempty"`
	ReadAccessKeyID     string `json:"read_access_key_id,omitempty"`
	ReadSecretAccessKey string `json:"read_secret_access_key,omitempty"`
}

// DefaultsConfig holds default settings.
type DefaultsConfig struct {
	Environment string `json:"environment"`
}

// HooksConfig holds hook commands.
type HooksConfig struct {
	PostPull string `json:"post_pull,omitempty"`
}

// LocalState represents ~/.envs3/state/<project>.json.
type LocalState struct {
	Environments      map[string]EnvState `json:"environments"`
	ActiveEnvironment string              `json:"active_environment"`
}

// EnvState tracks sync state for a single environment.
type EnvState struct {
	ETag      string   `json:"etag"`
	Version   int      `json:"version"`
	LastPull  string   `json:"last_pull"`
	KnownKeys []string `json:"known_keys"`
}

// AdminCredentials represents ~/.envs3/credentials/<project>.json.
// Deprecated: use .env.envs3 with ENVS3_WRITE_ACCESS_KEY_ID instead.
// Kept for backward compatibility.
type AdminCredentials struct {
	WriteAccessKeyID     string `json:"write_access_key_id"`
	WriteSecretAccessKey string `json:"write_secret_access_key"`
}
