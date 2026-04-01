package format

// KeyringFile represents keyring.json stored per environment in S3.
type KeyringFile struct {
	SchemaVersion int            `json:"schema_version"`
	DEKID         string         `json:"dek_id"`
	Algorithm     string         `json:"algorithm"`
	CreatedAt     string         `json:"created_at"`
	Wrapped       []WrappedEntry `json:"wrapped"`
}

// WrappedEntry contains a DEK encrypted for a specific user.
type WrappedEntry struct {
	Fingerprint string `json:"fingerprint"`
	WrappedDEK  string `json:"wrapped_dek"`
}
