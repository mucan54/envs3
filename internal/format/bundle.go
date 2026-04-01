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
}

// EncryptedSecret holds an AES-256-GCM encrypted value.
type EncryptedSecret struct {
	Value string `json:"value"`
	Nonce string `json:"nonce"`
	Tag   string `json:"tag"`
}
