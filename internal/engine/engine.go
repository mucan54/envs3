package engine

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/mucan54/envs3/internal/crypto"
	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
)

// Engine encapsulates all business logic for envs3 operations.
type Engine struct {
	Store   storage.Store
	Project string
}

// NewEngine creates a new engine instance.
func NewEngine(store storage.Store, project string) *Engine {
	return &Engine{Store: store, Project: project}
}

// getJSON fetches and unmarshals a JSON file from storage.
func (e *Engine) getJSON(ctx context.Context, key string, v interface{}) (string, error) {
	data, etag, err := e.Store.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return "", fmt.Errorf("parse %s: %w", key, err)
	}
	return etag, nil
}

// putJSON marshals and uploads a JSON file to storage.
func (e *Engine) putJSON(ctx context.Context, key string, v interface{}, ifMatchETag string) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return e.Store.Put(ctx, key, data, ifMatchETag)
}

// ResolveDEK unwraps the DEK for the given environment using the user's private key.
func (e *Engine) ResolveDEK(ctx context.Context, env string, priv, pub [32]byte) ([]byte, *format.KeyringFile, error) {
	var keyring format.KeyringFile
	_, err := e.getJSON(ctx, storage.KeyringPath(e.Project, env), &keyring)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch keyring: %w", err)
	}

	fp := crypto.Fingerprint(pub)
	for _, entry := range keyring.Wrapped {
		if entry.Fingerprint == fp {
			sealed, err := base64.StdEncoding.DecodeString(entry.WrappedDEK)
			if err != nil {
				return nil, nil, fmt.Errorf("decode wrapped DEK: %w", err)
			}
			dek, err := crypto.OpenBox(sealed, pub, priv)
			if err != nil {
				return nil, nil, fmt.Errorf("unwrap DEK: %w", err)
			}
			return dek, &keyring, nil
		}
	}

	return nil, nil, fmt.Errorf("access denied: your key is not authorized for '%s'", env)
}

// DecryptBundle decrypts all secrets in a bundle using the given DEK.
func DecryptBundle(bundle *format.Bundle, dek []byte) (map[string]string, error) {
	secrets := make(map[string]string, len(bundle.Secrets))
	for key, enc := range bundle.Secrets {
		plaintext, err := crypto.DecryptValue(dek, enc.Value, enc.Nonce, enc.Tag)
		if err != nil {
			return nil, fmt.Errorf("decrypt key '%s': %w", key, err)
		}
		secrets[key] = string(plaintext)
	}
	return secrets, nil
}

// EncryptSecrets encrypts a plaintext secrets map using the given DEK.
func EncryptSecrets(secrets map[string]string, dek []byte) (map[string]format.EncryptedSecret, error) {
	encrypted := make(map[string]format.EncryptedSecret, len(secrets))
	for key, value := range secrets {
		val, nonce, tag, err := crypto.EncryptValue(dek, []byte(value))
		if err != nil {
			return nil, fmt.Errorf("encrypt key '%s': %w", key, err)
		}
		encrypted[key] = format.EncryptedSecret{
			Value: val,
			Nonce: nonce,
			Tag:   tag,
		}
	}
	return encrypted, nil
}

// ComputeBundleChecksum computes the checksum for the secrets in a bundle.
func ComputeBundleChecksum(secrets map[string]format.EncryptedSecret) (string, error) {
	canonical, err := format.CanonicalJSON(secrets)
	if err != nil {
		return "", err
	}
	return crypto.ComputeChecksum(canonical), nil
}
