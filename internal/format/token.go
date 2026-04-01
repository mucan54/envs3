package format

// TokenPayload is the decoded content of a service token.
type TokenPayload struct {
	V         int          `json:"v"`
	Project   string       `json:"project"`
	Env       string       `json:"env"`
	Name      string       `json:"name"`
	Perm      string       `json:"perm"`
	DEK       string       `json:"dek"`
	DEKID     string       `json:"dek_id"`
	Storage   TokenStorage `json:"storage"`
	CreatedAt string       `json:"created_at"`
	CreatedBy string       `json:"created_by"`
	ExpiresAt *string      `json:"expires_at"`
}

// TokenStorage holds S3 credentials embedded in a token.
type TokenStorage struct {
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}
