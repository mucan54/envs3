package format

// ProjectConfig represents .envs3.json committed to the repository.
type ProjectConfig struct {
	SchemaVersion int            `json:"schema_version"`
	Project       string         `json:"project"`
	Storage       StorageConfig  `json:"storage"`
	Defaults      DefaultsConfig `json:"defaults"`
	Hooks         *HooksConfig   `json:"hooks,omitempty"`
}

// StorageConfig holds S3 connection details.
type StorageConfig struct {
	Type               string `json:"type"`
	Endpoint           string `json:"endpoint"`
	Bucket             string `json:"bucket"`
	Region             string `json:"region"`
	ReadAccessKeyID    string `json:"read_access_key_id"`
	ReadSecretAccessKey string `json:"read_secret_access_key"`
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
type AdminCredentials struct {
	WriteAccessKeyID     string `json:"write_access_key_id"`
	WriteSecretAccessKey string `json:"write_secret_access_key"`
}
