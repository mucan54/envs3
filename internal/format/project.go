package format

// ProjectFile represents project.json stored in S3.
type ProjectFile struct {
	SchemaVersion int      `json:"schema_version"`
	Name          string   `json:"name"`
	CreatedAt     string   `json:"created_at"`
	CreatedBy     string   `json:"created_by"`
	Environments  []string `json:"environments"`
}
