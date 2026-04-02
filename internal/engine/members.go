package engine

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/mucan54/envs3/internal/crypto"
	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
)

// AddMember adds a new member to the project and grants environment access.
func (e *Engine) AddMember(ctx context.Context, newPub [32]byte, email, role string, envs []string, adminPriv, adminPub [32]byte) error {
	var members format.MembersFile
	membersETag, err := e.getJSON(ctx, storage.MembersPath(e.Project), &members)
	if err != nil {
		return fmt.Errorf("fetch members: %w", err)
	}

	newFP := crypto.Fingerprint(newPub)

	// Check for duplicate
	for _, m := range members.Members {
		if m.Fingerprint == newFP {
			return fmt.Errorf("member with fingerprint %s already exists", newFP)
		}
	}

	// For each environment, wrap the DEK for the new member
	for _, envName := range envs {
		dek, keyring, err := e.ResolveDEK(ctx, envName, adminPriv, adminPub)
		if err != nil {
			return fmt.Errorf("resolve DEK for %s: %w", envName, err)
		}

		// Wrap DEK for new member
		sealed, err := crypto.SealBox(dek, newPub)
		if err != nil {
			return fmt.Errorf("wrap DEK for %s: %w", envName, err)
		}

		keyring.Wrapped = append(keyring.Wrapped, format.WrappedEntry{
			Fingerprint: newFP,
			WrappedDEK:  base64.StdEncoding.EncodeToString(sealed),
		})

		if _, err := e.putJSON(ctx, storage.KeyringPath(e.Project, envName), keyring, ""); err != nil {
			return fmt.Errorf("update keyring for %s: %w", envName, err)
		}
	}

	// Add to members.json
	now := time.Now().UTC().Format(time.RFC3339)
	members.Members = append(members.Members, format.Member{
		Email:        email,
		PublicKey:    crypto.EncodeKey(newPub),
		Fingerprint:  newFP,
		Role:         role,
		Environments: envs,
		AddedAt:      now,
	})

	if _, err := e.putJSON(ctx, storage.MembersPath(e.Project), &members, membersETag); err != nil {
		return fmt.Errorf("update members: %w", err)
	}

	// Audit log
	adminFP := crypto.Fingerprint(adminPub)
	e.AuditLog(ctx, &format.AuditEntry{
		Action:      "member_add",
		Actor:       email,
		Fingerprint: adminFP,
		Details: &format.AuditDetails{
			MemberEmail:  email,
			MemberRole:   role,
			Environments: envs,
		},
	})

	return nil
}

// RemoveMember removes a member from specified environments (or all) and rotates DEKs.
func (e *Engine) RemoveMember(ctx context.Context, email string, envs []string, adminPriv, adminPub [32]byte, adminEmail string) error {
	var members format.MembersFile
	membersETag, err := e.getJSON(ctx, storage.MembersPath(e.Project), &members)
	if err != nil {
		return fmt.Errorf("fetch members: %w", err)
	}

	// Find the member
	memberIdx := -1
	for i, m := range members.Members {
		if m.Email == email {
			memberIdx = i
			break
		}
	}
	if memberIdx < 0 {
		return fmt.Errorf("member %s not found", email)
	}

	member := members.Members[memberIdx]

	// Determine affected environments
	affectedEnvs := envs
	if len(affectedEnvs) == 0 {
		affectedEnvs = member.Environments
	}

	// Rotate DEK for each affected environment
	for _, envName := range affectedEnvs {
		if err := e.rotateDEK(ctx, envName, member.Fingerprint, adminPriv, adminPub, adminEmail, &members); err != nil {
			return fmt.Errorf("rotate DEK for %s: %w", envName, err)
		}
	}

	// Update member's environment list
	remaining := make([]string, 0)
	for _, env := range member.Environments {
		removed := false
		for _, ae := range affectedEnvs {
			if env == ae {
				removed = true
				break
			}
		}
		if !removed {
			remaining = append(remaining, env)
		}
	}

	if len(remaining) == 0 {
		// Remove member entirely
		members.Members = append(members.Members[:memberIdx], members.Members[memberIdx+1:]...)
	} else {
		members.Members[memberIdx].Environments = remaining
	}

	if _, err := e.putJSON(ctx, storage.MembersPath(e.Project), &members, membersETag); err != nil {
		return fmt.Errorf("update members: %w", err)
	}

	// Audit log
	e.AuditLog(ctx, &format.AuditEntry{
		Action: "member_remove",
		Actor:  adminEmail,
		Details: &format.AuditDetails{
			MemberEmail:  email,
			Environments: affectedEnvs,
		},
	})

	return nil
}

// rotateDEK generates a new DEK, re-encrypts all secrets, and re-seals for remaining members.
func (e *Engine) rotateDEK(ctx context.Context, env, removedFP string, adminPriv, adminPub [32]byte, adminEmail string, members *format.MembersFile) error {
	// Generate new DEK
	newDEK, err := crypto.GenerateDEK()
	if err != nil {
		return err
	}

	// Get current bundle
	var bundle format.Bundle
	currentETag, err := e.getJSON(ctx, storage.CurrentPath(e.Project, env), &bundle)
	if err != nil {
		return fmt.Errorf("fetch current: %w", err)
	}

	// Resolve old DEK
	oldDEK, keyring, err := e.ResolveDEK(ctx, env, adminPriv, adminPub)
	if err != nil {
		return fmt.Errorf("resolve old DEK: %w", err)
	}

	// Decrypt with old DEK
	secrets, err := DecryptBundle(&bundle, oldDEK)
	if err != nil {
		return err
	}

	// Re-encrypt with new DEK
	encrypted, err := EncryptSecrets(secrets, newDEK)
	if err != nil {
		return err
	}

	checksum, err := ComputeBundleChecksum(encrypted)
	if err != nil {
		return err
	}

	// Increment dek_id
	newDEKID := incrementDEKID(bundle.DEKID)
	now := time.Now().UTC().Format(time.RFC3339)

	// Re-seal new DEK for remaining members (exclude removed member)
	var newWrapped []format.WrappedEntry
	for _, m := range members.Members {
		if m.Fingerprint == removedFP {
			continue
		}
		// Check if this member has access to this env
		hasAccess := false
		for _, mEnv := range m.Environments {
			if mEnv == env {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			continue
		}

		pub, err := crypto.DecodeKey(m.PublicKey)
		if err != nil {
			continue
		}
		sealed, err := crypto.SealBox(newDEK, pub)
		if err != nil {
			continue
		}
		newWrapped = append(newWrapped, format.WrappedEntry{
			Fingerprint: m.Fingerprint,
			WrappedDEK:  base64.StdEncoding.EncodeToString(sealed),
		})
	}

	// Update keyring
	keyring.DEKID = newDEKID
	keyring.CreatedAt = now
	keyring.Wrapped = newWrapped
	if _, err := e.putJSON(ctx, storage.KeyringPath(e.Project, env), keyring, ""); err != nil {
		return fmt.Errorf("update keyring: %w", err)
	}

	// Write history
	newVersion := bundle.Version + 1
	history := format.HistoryEntry{
		SchemaVersion: 1,
		Environment:   env,
		Version:       bundle.Version,
		ParentVersion: bundle.Version - 1,
		DEKID:         bundle.DEKID,
		CreatedAt:     bundle.UpdatedAt,
		CreatedBy:     bundle.UpdatedBy,
		Checksum:      bundle.Checksum,
		Changes:       []format.Change{},
		Secrets:       bundle.Secrets,
	}
	if _, err := e.putJSON(ctx, storage.HistoryPath(e.Project, env, bundle.Version), &history, ""); err != nil {
		return fmt.Errorf("write history: %w", err)
	}

	// Write new current.json
	newBundle := format.Bundle{
		SchemaVersion: 1,
		Environment:   env,
		Version:       newVersion,
		DEKID:         newDEKID,
		UpdatedAt:     now,
		UpdatedBy:     adminEmail,
		Checksum:      checksum,
		Secrets:       encrypted,
	}
	if _, err := e.putJSON(ctx, storage.CurrentPath(e.Project, env), &newBundle, currentETag); err != nil {
		return fmt.Errorf("update current: %w", err)
	}

	return nil
}

func incrementDEKID(dekID string) string {
	// Format: dek_<env>_v<n>
	// Find the last 'v' and increment the number
	for i := len(dekID) - 1; i >= 0; i-- {
		if dekID[i] == 'v' {
			prefix := dekID[:i+1]
			var n int
			fmt.Sscanf(dekID[i+1:], "%d", &n)
			return fmt.Sprintf("%s%d", prefix, n+1)
		}
	}
	return dekID + "_v2"
}

// ListMembers returns all members.
func (e *Engine) ListMembers(ctx context.Context) ([]format.Member, error) {
	var members format.MembersFile
	if _, err := e.getJSON(ctx, storage.MembersPath(e.Project), &members); err != nil {
		return nil, err
	}
	return members.Members, nil
}
