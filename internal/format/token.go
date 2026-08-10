package format

// TokenPayload is the decoded content of a service token (v2).
// v2 tokens carry their own keypair instead of raw DEK.
// The token's public key is registered in the keyring like a member.
// This means tokens survive DEK rotation — when a member is removed,
// the token's wrapped DEK is updated alongside remaining members.
type TokenPayload struct {
	V          int          `json:"v"`
	Project    string       `json:"project"`
	Env        string       `json:"env"`
	Name       string       `json:"name"`
	Perm       string       `json:"perm"`
	PrivateKey string       `json:"private_key"` // v2: base64 X25519 private key (token unwraps DEK from keyring)
	PublicKey  string       `json:"public_key"`  // v2: base64 X25519 public key (registered in keyring)
	Storage    TokenStorage `json:"storage"`
	CreatedAt  string       `json:"created_at"`
	CreatedBy  string       `json:"created_by"`
	ExpiresAt  *string      `json:"expires_at"`
}

// TokenStorage holds S3 credentials embedded in a token.
type TokenStorage struct {
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket"`
	Region          string `json:"region,omitempty"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}
