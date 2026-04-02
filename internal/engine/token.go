package engine

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mucan54/envs3/internal/crypto"
	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
)

// CreateToken generates a service token with its own keypair.
// The token's public key is added to the keyring so it survives DEK rotation.
func (e *Engine) CreateToken(ctx context.Context, name, env, perm, email string, ttl time.Duration, s3Cfg storage.S3Config, priv, pub [32]byte) (string, error) {
	// Generate a dedicated keypair for this token
	tokenPub, tokenPriv, err := crypto.GenerateKeypair()
	if err != nil {
		return "", fmt.Errorf("generate token keypair: %w", err)
	}

	tokenFP := crypto.Fingerprint(tokenPub)

	// Resolve current DEK and wrap it for the token's public key
	dek, keyring, err := e.ResolveDEK(ctx, env, priv, pub)
	if err != nil {
		return "", err
	}

	sealed, err := crypto.SealBox(dek, tokenPub)
	if err != nil {
		return "", fmt.Errorf("wrap DEK for token: %w", err)
	}

	// Add token's public key to keyring
	keyring.Wrapped = append(keyring.Wrapped, format.WrappedEntry{
		Fingerprint: tokenFP,
		WrappedDEK:  base64.StdEncoding.EncodeToString(sealed),
	})
	if _, err := e.putJSON(ctx, storage.KeyringPath(e.Project, env), keyring, ""); err != nil {
		return "", fmt.Errorf("update keyring: %w", err)
	}

	// Build token payload (v2 — carries keypair, not raw DEK)
	now := time.Now().UTC()
	var expiresAt *string
	if ttl > 0 {
		exp := now.Add(ttl).Format(time.RFC3339)
		expiresAt = &exp
	}

	payload := format.TokenPayload{
		V:          2,
		Project:    e.Project,
		Env:        env,
		Name:       name,
		Perm:       perm,
		PrivateKey: crypto.EncodeKey(tokenPriv),
		PublicKey:  crypto.EncodeKey(tokenPub),
		Storage: format.TokenStorage{
			Endpoint:        s3Cfg.Endpoint,
			Bucket:          s3Cfg.Bucket,
			Region:          s3Cfg.Region,
			AccessKeyID:     s3Cfg.AccessKeyID,
			SecretAccessKey: s3Cfg.SecretAccessKey,
		},
		CreatedAt: now.Format(time.RFC3339),
		CreatedBy: email,
		ExpiresAt: expiresAt,
	}

	// Register token in members.json
	var members format.MembersFile
	membersETag, err := e.getJSON(ctx, storage.MembersPath(e.Project), &members)
	if err != nil {
		return "", fmt.Errorf("fetch members: %w", err)
	}

	members.Tokens = append(members.Tokens, format.TokenMeta{
		Name:        name,
		PublicKey:   crypto.EncodeKey(tokenPub),
		Fingerprint: tokenFP,
		Environment: env,
		Permissions: perm,
		CreatedBy:   email,
		CreatedAt:   now.Format(time.RFC3339),
		ExpiresAt:   expiresAt,
	})

	if _, err := e.putJSON(ctx, storage.MembersPath(e.Project), &members, membersETag); err != nil {
		return "", fmt.Errorf("update members: %w", err)
	}

	// Audit log
	e.AuditLog(ctx, &format.AuditEntry{
		Action:      "token_create",
		Actor:       email,
		Environment: env,
		Details:     &format.AuditDetails{TokenName: name},
	})

	data, err := json.Marshal(&payload)
	if err != nil {
		return "", err
	}

	return "envs3_v1_" + base64.URLEncoding.EncodeToString(data), nil
}

// ListTokens returns all token metadata.
func (e *Engine) ListTokens(ctx context.Context) ([]format.TokenMeta, error) {
	var members format.MembersFile
	if _, err := e.getJSON(ctx, storage.MembersPath(e.Project), &members); err != nil {
		return nil, err
	}
	return members.Tokens, nil
}

// RevokeToken revokes a token: removes from keyring and rotates DEK.
func (e *Engine) RevokeToken(ctx context.Context, name string, adminPriv, adminPub [32]byte, adminEmail string) error {
	var members format.MembersFile
	membersETag, err := e.getJSON(ctx, storage.MembersPath(e.Project), &members)
	if err != nil {
		return fmt.Errorf("fetch members: %w", err)
	}

	tokenIdx := -1
	var tokenEnv, tokenFP string
	for i, t := range members.Tokens {
		if t.Name == name {
			tokenIdx = i
			tokenEnv = t.Environment
			tokenFP = t.Fingerprint
			break
		}
	}
	if tokenIdx < 0 {
		return fmt.Errorf("token '%s' not found", name)
	}

	// Remove token from members.json
	members.Tokens = append(members.Tokens[:tokenIdx], members.Tokens[tokenIdx+1:]...)

	// Rotate DEK — this removes the token's fingerprint from the keyring
	// and re-seals for remaining members + remaining tokens
	if err := e.rotateDEK(ctx, tokenEnv, tokenFP, adminPriv, adminPub, adminEmail, &members); err != nil {
		return fmt.Errorf("rotate DEK: %w", err)
	}

	if _, err := e.putJSON(ctx, storage.MembersPath(e.Project), &members, membersETag); err != nil {
		return fmt.Errorf("update members: %w", err)
	}

	// Audit log
	e.AuditLog(ctx, &format.AuditEntry{
		Action:      "token_revoke",
		Actor:       adminEmail,
		Environment: tokenEnv,
		Details:     &format.AuditDetails{TokenName: name},
	})

	return nil
}
