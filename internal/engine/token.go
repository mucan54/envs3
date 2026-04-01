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

// CreateToken generates a service token for CI/CD.
func (e *Engine) CreateToken(ctx context.Context, name, env, perm, email string, ttl time.Duration, s3Cfg storage.S3Config, priv, pub [32]byte) (string, error) {
	dek, _, err := e.ResolveDEK(ctx, env, priv, pub)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	var expiresAt *string
	if ttl > 0 {
		exp := now.Add(ttl).Format(time.RFC3339)
		expiresAt = &exp
	}

	var keyring format.KeyringFile
	if _, err := e.getJSON(ctx, storage.KeyringPath(e.Project, env), &keyring); err != nil {
		return "", fmt.Errorf("fetch keyring: %w", err)
	}

	payload := format.TokenPayload{
		V:       1,
		Project: e.Project,
		Env:     env,
		Name:    name,
		Perm:    perm,
		DEK:     base64.StdEncoding.EncodeToString(dek),
		DEKID:   keyring.DEKID,
		Storage: format.TokenStorage{
			Endpoint:        s3Cfg.Endpoint,
			Bucket:          s3Cfg.Bucket,
			AccessKeyID:     s3Cfg.AccessKeyID,
			SecretAccessKey: s3Cfg.SecretAccessKey,
		},
		CreatedAt: now.Format(time.RFC3339),
		CreatedBy: email,
		ExpiresAt: expiresAt,
	}

	tokenFP := crypto.Fingerprint(pub)

	var members format.MembersFile
	membersETag, err := e.getJSON(ctx, storage.MembersPath(e.Project), &members)
	if err != nil {
		return "", fmt.Errorf("fetch members: %w", err)
	}

	members.Tokens = append(members.Tokens, format.TokenMeta{
		Name:        name,
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

// RevokeToken revokes a token and rotates the DEK for its environment.
func (e *Engine) RevokeToken(ctx context.Context, name string, adminPriv, adminPub [32]byte, adminEmail string) error {
	var members format.MembersFile
	membersETag, err := e.getJSON(ctx, storage.MembersPath(e.Project), &members)
	if err != nil {
		return fmt.Errorf("fetch members: %w", err)
	}

	tokenIdx := -1
	var tokenEnv string
	for i, t := range members.Tokens {
		if t.Name == name {
			tokenIdx = i
			tokenEnv = t.Environment
			break
		}
	}
	if tokenIdx < 0 {
		return fmt.Errorf("token '%s' not found", name)
	}

	members.Tokens = append(members.Tokens[:tokenIdx], members.Tokens[tokenIdx+1:]...)

	if err := e.rotateDEK(ctx, tokenEnv, "", adminPriv, adminPub, adminEmail, &members); err != nil {
		return fmt.Errorf("rotate DEK: %w", err)
	}

	if _, err := e.putJSON(ctx, storage.MembersPath(e.Project), &members, membersETag); err != nil {
		return fmt.Errorf("update members: %w", err)
	}

	return nil
}
