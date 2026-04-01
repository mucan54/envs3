package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
)

// Rollback restores an environment to a previous version.
func (e *Engine) Rollback(ctx context.Context, env, email string, targetVersion int, priv, pub [32]byte) (*PushResult, error) {
	// Fetch the target history entry
	var target format.HistoryEntry
	_, err := e.getJSON(ctx, storage.HistoryPath(e.Project, env, targetVersion), &target)
	if err != nil {
		return nil, fmt.Errorf("fetch history v%d: %w", targetVersion, err)
	}

	// Fetch current bundle
	var current format.Bundle
	currentETag, err := e.getJSON(ctx, storage.CurrentPath(e.Project, env), &current)
	if err != nil {
		return nil, fmt.Errorf("fetch current: %w", err)
	}

	// Resolve DEK — we need the current DEK to decrypt for comparison
	dek, _, err := e.ResolveDEK(ctx, env, priv, pub)
	if err != nil {
		return nil, err
	}

	// Decrypt both for change computation
	currentSecrets, err := DecryptBundle(&current, dek)
	if err != nil {
		return nil, err
	}

	targetSecrets, err := DecryptBundle(&format.Bundle{Secrets: target.Secrets}, dek)
	if err != nil {
		return nil, fmt.Errorf("decrypt target: %w", err)
	}

	changes := computeChanges(currentSecrets, targetSecrets)

	now := time.Now().UTC().Format(time.RFC3339)
	newVersion := current.Version + 1

	// Re-encrypt target secrets with fresh nonces
	encrypted, err := EncryptSecrets(targetSecrets, dek)
	if err != nil {
		return nil, err
	}

	checksum, err := ComputeBundleChecksum(encrypted)
	if err != nil {
		return nil, err
	}

	// Write history for current version
	history := format.HistoryEntry{
		SchemaVersion: 1,
		Environment:   env,
		Version:       current.Version,
		ParentVersion: current.Version - 1,
		DEKID:         current.DEKID,
		CreatedAt:     current.UpdatedAt,
		CreatedBy:     current.UpdatedBy,
		Checksum:      current.Checksum,
		Changes:       changes,
		Secrets:       current.Secrets,
	}
	if _, err := e.putJSON(ctx, storage.HistoryPath(e.Project, env, current.Version), &history, ""); err != nil {
		return nil, fmt.Errorf("write history: %w", err)
	}

	// Write new current.json with rolled-back secrets
	bundle := format.Bundle{
		SchemaVersion: 1,
		Environment:   env,
		Version:       newVersion,
		DEKID:         current.DEKID,
		UpdatedAt:     now,
		UpdatedBy:     email,
		Checksum:      checksum,
		Secrets:       encrypted,
	}
	if _, err := e.putJSON(ctx, storage.CurrentPath(e.Project, env), &bundle, currentETag); err != nil {
		return nil, fmt.Errorf("rollback: %w", err)
	}

	return &PushResult{
		OldVersion: current.Version,
		NewVersion: newVersion,
		Changes:    changes,
	}, nil
}
