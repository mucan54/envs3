package format

// MembersFile represents members.json stored in S3.
type MembersFile struct {
	SchemaVersion int         `json:"schema_version"`
	Members       []Member    `json:"members"`
	Tokens        []TokenMeta `json:"tokens"`
}

// Member represents a project member.
type Member struct {
	Email        string   `json:"email"`
	PublicKey    string   `json:"public_key"`
	Fingerprint  string   `json:"fingerprint"`
	Role         string   `json:"role"`
	Environments []string `json:"environments"`
	AddedAt      string   `json:"added_at"`
}

// TokenMeta represents token metadata in members.json.
type TokenMeta struct {
	Name        string  `json:"name"`
	PublicKey   string  `json:"public_key"`   // token's X25519 public key (for DEK re-sealing on rotation)
	Fingerprint string  `json:"fingerprint"`
	Environment string  `json:"environment"`
	Permissions string  `json:"permissions"`
	CreatedBy   string  `json:"created_by"`
	CreatedAt   string  `json:"created_at"`
	ExpiresAt   *string `json:"expires_at"`
}
