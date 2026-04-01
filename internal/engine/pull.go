package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
)

// PullResult contains the result of a pull operation.
type PullResult struct {
	Secrets     map[string]string
	Environment string
	Version     int
	UpdatedAt   string
	ETag        string
	UpToDate    bool
}

// Pull fetches and decrypts secrets for an environment.
func (e *Engine) Pull(ctx context.Context, env string, priv, pub [32]byte, cachedETag string) (*PullResult, error) {
	// ETag check — if cachedETag is provided, do a HEAD first
	if cachedETag != "" {
		currentETag, err := e.Store.Head(ctx, storage.CurrentPath(e.Project, env))
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return nil, fmt.Errorf("check updates: %w", err)
		}
		if currentETag == cachedETag {
			return &PullResult{UpToDate: true, Environment: env}, nil
		}
	}

	// Full pull
	var bundle format.Bundle
	etag, err := e.getJSON(ctx, storage.CurrentPath(e.Project, env), &bundle)
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

	// Verify checksum
	checksum, err := ComputeBundleChecksum(bundle.Secrets)
	if err != nil {
		return nil, err
	}
	if checksum != bundle.Checksum {
		return nil, fmt.Errorf("checksum verification failed: bundle may be corrupted")
	}

	return &PullResult{
		Secrets:     secrets,
		Environment: bundle.Environment,
		Version:     bundle.Version,
		UpdatedAt:   bundle.UpdatedAt,
		ETag:        etag,
	}, nil
}
