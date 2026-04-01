package engine

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/mucan54/envs3/internal/crypto"
	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
)

// InitParams holds parameters for project initialization.
type InitParams struct {
	ProjectName  string
	Email        string
	Environments []string
	ImportEnv    string            // environment name to import into
	ImportData   map[string]string // key-value pairs to import
	PublicKey    [32]byte
}

// Init creates a new project in storage.
func (e *Engine) Init(ctx context.Context, params InitParams) error {
	now := time.Now().UTC().Format(time.RFC3339)

	// Create project.json
	project := format.ProjectFile{
		SchemaVersion: 1,
		Name:          params.ProjectName,
		CreatedAt:     now,
		CreatedBy:     params.Email,
		Environments:  params.Environments,
	}
	if _, err := e.putJSON(ctx, storage.ProjectPath(e.Project), &project, ""); err != nil {
		return err
	}

	fp := crypto.Fingerprint(params.PublicKey)

	// Create members.json
	members := format.MembersFile{
		SchemaVersion: 1,
		Members: []format.Member{
			{
				Email:        params.Email,
				PublicKey:    crypto.EncodeKey(params.PublicKey),
				Fingerprint:  fp,
				Role:         "admin",
				Environments: params.Environments,
				AddedAt:      now,
			},
		},
		Tokens: []format.TokenMeta{},
	}
	if _, err := e.putJSON(ctx, storage.MembersPath(e.Project), &members, ""); err != nil {
		return err
	}

	// Create each environment
	for _, envName := range params.Environments {
		dek, err := crypto.GenerateDEK()
		if err != nil {
			return err
		}

		// Wrap DEK for admin
		sealed, err := crypto.SealBox(dek, params.PublicKey)
		if err != nil {
			return err
		}

		dekID := "dek_" + envName + "_v1"

		keyring := format.KeyringFile{
			SchemaVersion: 1,
			DEKID:         dekID,
			Algorithm:     "x25519-xsalsa20-poly1305",
			CreatedAt:     now,
			Wrapped: []format.WrappedEntry{
				{
					Fingerprint: fp,
					WrappedDEK:  base64.StdEncoding.EncodeToString(sealed),
				},
			},
		}
		if _, err := e.putJSON(ctx, storage.KeyringPath(e.Project, envName), &keyring, ""); err != nil {
			return err
		}

		// Determine secrets to import
		secrets := make(map[string]string)
		if envName == params.ImportEnv && params.ImportData != nil {
			secrets = params.ImportData
		}

		// Encrypt and create current.json
		encrypted, err := EncryptSecrets(secrets, dek)
		if err != nil {
			return err
		}

		checksum, err := ComputeBundleChecksum(encrypted)
		if err != nil {
			return err
		}

		bundle := format.Bundle{
			SchemaVersion: 1,
			Environment:   envName,
			Version:       1,
			DEKID:         dekID,
			UpdatedAt:     now,
			UpdatedBy:     params.Email,
			Checksum:      checksum,
			Secrets:       encrypted,
		}
		if _, err := e.putJSON(ctx, storage.CurrentPath(e.Project, envName), &bundle, ""); err != nil {
			return err
		}

		// Write initial history entry
		history := format.HistoryEntry{
			SchemaVersion: 1,
			Environment:   envName,
			Version:       1,
			ParentVersion: 0,
			DEKID:         dekID,
			CreatedAt:     now,
			CreatedBy:     params.Email,
			Checksum:      checksum,
			Changes:       makeInitialChanges(secrets),
			Secrets:       encrypted,
		}
		if _, err := e.putJSON(ctx, storage.HistoryPath(e.Project, envName, 1), &history, ""); err != nil {
			return err
		}
	}

	return nil
}

func makeInitialChanges(secrets map[string]string) []format.Change {
	changes := make([]format.Change, 0, len(secrets))
	for key := range secrets {
		changes = append(changes, format.Change{Key: key, Action: "added"})
	}
	return changes
}
