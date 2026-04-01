package engine

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
)

// PushResult contains the result of a push operation.
type PushResult struct {
	OldVersion int
	NewVersion int
	Changes    []format.Change
}

// Push encrypts and uploads local secrets to the active environment.
func (e *Engine) Push(ctx context.Context, env, email string, localSecrets map[string]string, priv, pub [32]byte) (*PushResult, error) {
	// Get current bundle for comparison
	var currentBundle format.Bundle
	currentETag, err := e.getJSON(ctx, storage.CurrentPath(e.Project, env), &currentBundle)
	if err != nil {
		return nil, fmt.Errorf("fetch current: %w", err)
	}

	dek, _, err := e.ResolveDEK(ctx, env, priv, pub)
	if err != nil {
		return nil, err
	}

	// Decrypt current secrets for diff
	currentSecrets, err := DecryptBundle(&currentBundle, dek)
	if err != nil {
		return nil, err
	}

	// Compute changes
	changes := computeChanges(currentSecrets, localSecrets)
	if len(changes) == 0 {
		return nil, nil // nothing to push
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Encrypt all values with fresh nonces
	encrypted, err := EncryptSecrets(localSecrets, dek)
	if err != nil {
		return nil, err
	}

	checksum, err := ComputeBundleChecksum(encrypted)
	if err != nil {
		return nil, err
	}

	newVersion := currentBundle.Version + 1

	// Write history entry (copy of previous current + changes)
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

	// Write new current.json with conditional write
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
		return nil, fmt.Errorf("push: %w", err)
	}

	return &PushResult{
		OldVersion: currentBundle.Version,
		NewVersion: newVersion,
		Changes:    changes,
	}, nil
}

func computeChanges(current, local map[string]string) []format.Change {
	var changes []format.Change

	// Check for added and updated
	keys := make([]string, 0, len(local))
	for k := range local {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if oldVal, exists := current[k]; !exists {
			changes = append(changes, format.Change{Key: k, Action: "added"})
		} else if oldVal != local[k] {
			changes = append(changes, format.Change{Key: k, Action: "updated"})
		}
	}

	// Check for removed
	for k := range current {
		if _, exists := local[k]; !exists {
			changes = append(changes, format.Change{Key: k, Action: "removed"})
		}
	}

	return changes
}
