package engine

import (
	"context"
	"fmt"
	"sort"

	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
)

// ViewResult contains the result of a view operation.
type ViewResult struct {
	Secrets     map[string]string
	Keys        []string
	Environment string
	Version     int
}

// View fetches and decrypts secrets without modifying any local files.
func (e *Engine) View(ctx context.Context, env string, priv, pub [32]byte) (*ViewResult, error) {
	var bundle format.Bundle
	_, err := e.getJSON(ctx, storage.CurrentPath(e.Project, env), &bundle)
	if err != nil {
		return nil, fmt.Errorf("fetch secrets: %w", err)
	}

	dek, _, err := e.ResolveDEK(ctx, env, priv, pub)
	if err != nil {
		return nil, err
	}

	secrets, err := DecryptBundle(&bundle, dek)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(secrets))
	for k := range secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return &ViewResult{
		Secrets:     secrets,
		Keys:        keys,
		Environment: env,
		Version:     bundle.Version,
	}, nil
}

// ViewKeys returns only the key names from an environment (no DEK needed).
func (e *Engine) ViewKeys(ctx context.Context, env string) ([]string, int, error) {
	var bundle format.Bundle
	_, err := e.getJSON(ctx, storage.CurrentPath(e.Project, env), &bundle)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch secrets: %w", err)
	}

	keys := make([]string, 0, len(bundle.Secrets))
	for k := range bundle.Secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys, bundle.Version, nil
}
