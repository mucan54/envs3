package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
)

// SetResult contains the result of a set/unset operation.
type SetResult struct {
	OldVersion int
	NewVersion int
	Changes    []format.Change
}

// Set modifies secrets directly on a remote environment without touching local .env.
func (e *Engine) Set(ctx context.Context, env, email string, setKVs map[string]string, unsetKeys []string, priv, pub [32]byte) (*SetResult, error) {
	var currentBundle format.Bundle
	currentETag, err := e.getJSON(ctx, storage.CurrentPath(e.Project, env), &currentBundle)
	if err != nil {
		return nil, fmt.Errorf("fetch current: %w", err)
	}

	dek, _, err := e.ResolveDEK(ctx, env, priv, pub)
	if err != nil {
		return nil, err
	}

	currentSecrets, err := DecryptBundle(&currentBundle, dek)
	if err != nil {
		return nil, err
	}

	// Apply changes in memory
	var changes []format.Change
	for key, value := range setKVs {
		if _, exists := currentSecrets[key]; exists {
			changes = append(changes, format.Change{Key: key, Action: "updated"})
		} else {
			changes = append(changes, format.Change{Key: key, Action: "added"})
		}
		currentSecrets[key] = value
	}
	for _, key := range unsetKeys {
		if _, exists := currentSecrets[key]; exists {
			delete(currentSecrets, key)
			changes = append(changes, format.Change{Key: key, Action: "removed"})
		}
	}

	if len(changes) == 0 {
		return nil, nil
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Re-encrypt all values
	encrypted, err := EncryptSecrets(currentSecrets, dek)
	if err != nil {
		return nil, err
	}

	checksum, err := ComputeBundleChecksum(encrypted)
	if err != nil {
		return nil, err
	}

	newVersion := currentBundle.Version + 1

	// Write history
	history := format.HistoryEntry{
		SchemaVersion: 1,
		Environment:   env,
		Version:       currentBundle.Version,
		ParentVersion: currentBundle.Version - 1,
		DEKID:         currentBundle.DEKID,
		CreatedAt:     currentBundle.UpdatedAt,
		CreatedBy:     currentBundle.UpdatedBy,
		Checksum:      currentBundle.Checksum,
		Changes:       changes,
		Secrets:       currentBundle.Secrets,
	}
	if _, err := e.putJSON(ctx, storage.HistoryPath(e.Project, env, currentBundle.Version), &history, ""); err != nil {
		return nil, fmt.Errorf("write history: %w", err)
	}

	// Write new current.json
	bundle := format.Bundle{
		SchemaVersion: 1,
		Environment:   env,
		Version:       newVersion,
		DEKID:         currentBundle.DEKID,
		UpdatedAt:     now,
		UpdatedBy:     email,
		Checksum:      checksum,
		Secrets:       encrypted,
	}
	if _, err := e.putJSON(ctx, storage.CurrentPath(e.Project, env), &bundle, currentETag); err != nil {
		return nil, fmt.Errorf("set: %w", err)
	}

	return &SetResult{
		OldVersion: currentBundle.Version,
		NewVersion: newVersion,
		Changes:    changes,
	}, nil
}
