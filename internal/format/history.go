package format

// HistoryEntry represents a history/<version>.json file.
type HistoryEntry struct {
	SchemaVersion int                        `json:"schema_version"`
	Environment   string                     `json:"environment"`
	Version       int                        `json:"version"`
	ParentVersion int                        `json:"parent_version"`
	DEKID         string                     `json:"dek_id"`
	CreatedAt     string                     `json:"created_at"`
	CreatedBy     string                     `json:"created_by"`
	Checksum      string                     `json:"checksum"`
	Changes       []Change                   `json:"changes"`
	Secrets       map[string]EncryptedSecret `json:"secrets"`
}

// Change describes a single key change in a history entry.
type Change struct {
	Key    string `json:"key"`
	Action string `json:"action"` // "added", "updated", "removed"
}
