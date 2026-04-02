package format

// AuditEntry represents a single audit log entry stored in S3.
// Path: <project>/audit/<timestamp>_<action>.json
type AuditEntry struct {
	SchemaVersion int    `json:"schema_version"`
	Timestamp     string `json:"timestamp"`
	Action        string `json:"action"` // "pull", "push", "set", "unset", "view", "member_add", "member_remove", "token_create", "token_revoke", "rollback", "env_create"
	Actor         string `json:"actor"`  // email or token name
	Fingerprint   string `json:"fingerprint"`
	Environment   string `json:"environment,omitempty"`
	Details       *AuditDetails `json:"details,omitempty"`
}

// AuditDetails holds action-specific information.
type AuditDetails struct {
	// For push/set: which keys changed
	KeysAdded   []string `json:"keys_added,omitempty"`
	KeysUpdated []string `json:"keys_updated,omitempty"`
	KeysRemoved []string `json:"keys_removed,omitempty"`

	// For version changes
	FromVersion int `json:"from_version,omitempty"`
	ToVersion   int `json:"to_version,omitempty"`

	// For member operations
	MemberEmail string `json:"member_email,omitempty"`
	MemberRole  string `json:"member_role,omitempty"`
	Environments []string `json:"environments,omitempty"`

	// For token operations
	TokenName   string `json:"token_name,omitempty"`

	// For rotation
	OldDEKID string `json:"old_dek_id,omitempty"`
	NewDEKID string `json:"new_dek_id,omitempty"`
}
