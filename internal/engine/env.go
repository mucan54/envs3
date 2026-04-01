package engine

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/mucan54/envs3/internal/crypto"
	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
)

// CreateEnv creates a new environment.
func (e *Engine) CreateEnv(ctx context.Context, name, email string, fromEnv string, priv, pub [32]byte) error {
	// Update project.json
	var project format.ProjectFile
	projectETag, err := e.getJSON(ctx, storage.ProjectPath(e.Project), &project)
	if err != nil {
		return fmt.Errorf("fetch project: %w", err)
	}

	for _, env := range project.Environments {
		if env == name {
			return fmt.Errorf("environment '%s' already exists", name)
		}
	}

	project.Environments = append(project.Environments, name)

	// Generate new DEK
	newDEK, err := crypto.GenerateDEK()
	if err != nil {
		return err
	}

	fp := crypto.Fingerprint(pub)
	now := time.Now().UTC().Format(time.RFC3339)
	dekID := "dek_" + name + "_v1"

	// Wrap DEK for admin
	sealed, err := crypto.SealBox(newDEK, pub)
	if err != nil {
		return err
	}

	keyring := format.KeyringFile{
		SchemaVersion: 1,
		DEKID:         dekID,
		Algorithm:     "x25519-xsalsa20-poly1305",
		CreatedAt:     now,
		Wrapped: []format.WrappedEntry{
			{Fingerprint: fp, WrappedDEK: base64.StdEncoding.EncodeToString(sealed)},
		},
	}

	// Get secrets (copy from source env or empty)
	secrets := make(map[string]string)
	if fromEnv != "" {
		srcDEK, _, err := e.ResolveDEK(ctx, fromEnv, priv, pub)
		if err != nil {
			return fmt.Errorf("resolve source DEK: %w", err)
		}

		var srcBundle format.Bundle
		if _, err := e.getJSON(ctx, storage.CurrentPath(e.Project, fromEnv), &srcBundle); err != nil {
			return fmt.Errorf("fetch source: %w", err)
		}

		secrets, err = DecryptBundle(&srcBundle, srcDEK)
		if err != nil {
			return fmt.Errorf("decrypt source: %w", err)
		}
	}

	encrypted, err := EncryptSecrets(secrets, newDEK)
	if err != nil {
		return err
	}

	checksum, err := ComputeBundleChecksum(encrypted)
	if err != nil {
		return err
	}

	bundle := format.Bundle{
		SchemaVersion: 1,
		Environment:   name,
		Version:       1,
		DEKID:         dekID,
		UpdatedAt:     now,
		UpdatedBy:     email,
		Checksum:      checksum,
		Secrets:       encrypted,
	}

	// Write all files
	if _, err := e.putJSON(ctx, storage.KeyringPath(e.Project, name), &keyring, ""); err != nil {
		return err
	}
	if _, err := e.putJSON(ctx, storage.CurrentPath(e.Project, name), &bundle, ""); err != nil {
		return err
	}
	if _, err := e.putJSON(ctx, storage.ProjectPath(e.Project), &project, projectETag); err != nil {
		return err
	}

	// Write initial history
	history := format.HistoryEntry{
		SchemaVersion: 1,
		Environment:   name,
		Version:       1,
		ParentVersion: 0,
		DEKID:         dekID,
		CreatedAt:     now,
		CreatedBy:     email,
		Checksum:      checksum,
		Changes:       makeInitialChanges(secrets),
		Secrets:       encrypted,
	}
	if _, err := e.putJSON(ctx, storage.HistoryPath(e.Project, name, 1), &history, ""); err != nil {
		return err
	}

	return nil
}

// ListEnvs returns the list of environments.
func (e *Engine) ListEnvs(ctx context.Context) ([]string, error) {
	var project format.ProjectFile
	if _, err := e.getJSON(ctx, storage.ProjectPath(e.Project), &project); err != nil {
		// Fallback: list from S3 prefix
		keys, err := e.Store.List(ctx, storage.EnvironmentsPrefix(e.Project))
		if err != nil {
			return nil, err
		}
		envMap := make(map[string]bool)
		prefix := storage.EnvironmentsPrefix(e.Project)
		for _, k := range keys {
			rel := strings.TrimPrefix(k, prefix)
			parts := strings.SplitN(rel, "/", 2)
			if len(parts) > 0 && parts[0] != "" {
				envMap[parts[0]] = true
			}
		}
		envs := make([]string, 0, len(envMap))
		for env := range envMap {
			envs = append(envs, env)
		}
		return envs, nil
	}
	return project.Environments, nil
}
