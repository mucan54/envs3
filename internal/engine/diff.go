package engine

import (
	"context"
	"fmt"
	"sort"

	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
)

// DiffEntry represents a single key difference between environments.
type DiffEntry struct {
	Key    string
	Left   string
	Right  string
	Status string // "identical", "changed", "left_only", "right_only"
}

// Diff compares two environments and returns differences.
func (e *Engine) Diff(ctx context.Context, env1, env2 string, priv, pub [32]byte) ([]DiffEntry, error) {
	secrets1, err := e.decryptEnv(ctx, env1, priv, pub)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", env1, err)
	}

	secrets2, err := e.decryptEnv(ctx, env2, priv, pub)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", env2, err)
	}

	return diffMaps(secrets1, secrets2), nil
}

// DiffLocal compares local secrets against a remote environment.
func (e *Engine) DiffLocal(ctx context.Context, env string, localSecrets map[string]string, priv, pub [32]byte) ([]DiffEntry, error) {
	remote, err := e.decryptEnv(ctx, env, priv, pub)
	if err != nil {
		return nil, err
	}
	return diffMaps(localSecrets, remote), nil
}

func (e *Engine) decryptEnv(ctx context.Context, env string, priv, pub [32]byte) (map[string]string, error) {
	var bundle format.Bundle
	_, err := e.getJSON(ctx, storage.CurrentPath(e.Project, env), &bundle)
	if err != nil {
		return nil, err
	}

	dek, _, err := e.ResolveDEK(ctx, env, priv, pub)
	if err != nil {
		return nil, err
	}

	return DecryptBundle(&bundle, dek)
}

func diffMaps(left, right map[string]string) []DiffEntry {
	allKeys := make(map[string]bool)
	for k := range left {
		allKeys[k] = true
	}
	for k := range right {
		allKeys[k] = true
	}

	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var entries []DiffEntry
	for _, k := range keys {
		lv, lOk := left[k]
		rv, rOk := right[k]

		var entry DiffEntry
		entry.Key = k
		entry.Left = lv
		entry.Right = rv

		switch {
		case lOk && rOk && lv == rv:
			entry.Status = "identical"
		case lOk && rOk:
			entry.Status = "changed"
		case lOk:
			entry.Status = "left_only"
		default:
			entry.Status = "right_only"
		}
		entries = append(entries, entry)
	}

	return entries
}
