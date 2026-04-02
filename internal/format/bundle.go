package format

// Bundle represents current.json — the encrypted secrets for an environment.
type Bundle struct {
	SchemaVersion int                        `json:"schema_version"`
	Environment   string                     `json:"environment"`
	Version       int                        `json:"version"`
	DEKID         string                     `json:"dek_id"`
	UpdatedAt     string                     `json:"updated_at"`
	UpdatedBy     string                     `json:"updated_by"`
	Checksum      string                     `json:"checksum"`
	Secrets       map[string]EncryptedSecret `json:"secrets"`
	Metadata      map[string]SecretMetadata  `json:"metadata,omitempty"` // per-key metadata (OWASP compliance)
}

// EncryptedSecret holds an AES-256-GCM encrypted value.
type EncryptedSecret struct {
	Value string `json:"value"`
	Nonce string `json:"nonce"`
	Tag   string `json:"tag"`
}

// SecretMetadata holds per-key metadata for OWASP compliance.
// Key names in this map match key names in Secrets.
type SecretMetadata struct {
	// Rotation tracking (OWASP: credentials should be rotated periodically)
	RotatedAt    string `json:"rotated_at,omitempty"`     // when this secret was last rotated
	RotationDays int    `json:"rotation_days,omitempty"`  // expected rotation interval in days (0 = no policy)

	// Expiry (OWASP: secrets should have limited lifetime)
	ExpiresAt string `json:"expires_at,omitempty"` // ISO 8601 expiry timestamp

	// Tags for organization and filtering
	Tags []string `json:"tags,omitempty"` // e.g. ["database", "critical", "pci"]

	// Description
	Description string `json:"description,omitempty"` // human-readable description
}
