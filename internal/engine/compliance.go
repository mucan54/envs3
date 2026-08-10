package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
)

// ComplianceIssue represents a single OWASP compliance warning.
type ComplianceIssue struct {
	Severity    string // "critical", "warning", "info"
	Environment string
	Key         string
	Rule        string
	Message     string
}

// CheckCompliance checks OWASP secrets management compliance for an environment.
func (e *Engine) CheckCompliance(ctx context.Context, env string, priv, pub [32]byte) ([]ComplianceIssue, error) {
	var issues []ComplianceIssue
	now := time.Now().UTC()

	// Load bundle
	var bundle format.Bundle
	_, err := e.getJSON(ctx, storage.CurrentPath(e.Project, env), &bundle)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", env, err)
	}

	// Check 1: Encryption at rest (always passes — envs3 encrypts by design)
	// No issue to report

	// Check 2: Secret rotation — warn if secrets haven't been rotated
	for key, meta := range bundle.Metadata {
		if meta.RotationDays > 0 && meta.RotatedAt != "" {
			rotatedAt, err := time.Parse(time.RFC3339, meta.RotatedAt)
			if err == nil {
				daysSince := int(now.Sub(rotatedAt).Hours() / 24)
				if daysSince > meta.RotationDays {
					issues = append(issues, ComplianceIssue{
						Severity:    "critical",
						Environment: env,
						Key:         key,
						Rule:        "rotation-overdue",
						Message:     fmt.Sprintf("last rotated %d days ago (policy: every %d days)", daysSince, meta.RotationDays),
					})
				} else if daysSince > meta.RotationDays*3/4 {
					issues = append(issues, ComplianceIssue{
						Severity:    "warning",
						Environment: env,
						Key:         key,
						Rule:        "rotation-approaching",
						Message:     fmt.Sprintf("last rotated %d days ago (policy: every %d days)", daysSince, meta.RotationDays),
					})
				}
			}
		}

		// Check 3: Secret expiry
		if meta.ExpiresAt != "" {
			expiresAt, err := time.Parse(time.RFC3339, meta.ExpiresAt)
			if err == nil {
				if now.After(expiresAt) {
					issues = append(issues, ComplianceIssue{
						Severity:    "critical",
						Environment: env,
						Key:         key,
						Rule:        "expired",
						Message:     fmt.Sprintf("expired at %s", meta.ExpiresAt),
					})
				} else if expiresAt.Sub(now).Hours() < 7*24 {
					issues = append(issues, ComplianceIssue{
						Severity:    "warning",
						Environment: env,
						Key:         key,
						Rule:        "expiring-soon",
						Message:     fmt.Sprintf("expires in %d days", int(expiresAt.Sub(now).Hours()/24)),
					})
				}
			}
		}
	}

	// Check 4: Secrets without rotation policy
	for key := range bundle.Secrets {
		meta, hasMeta := bundle.Metadata[key]
		if !hasMeta || meta.RotationDays == 0 {
			issues = append(issues, ComplianceIssue{
				Severity:    "info",
				Environment: env,
				Key:         key,
				Rule:        "no-rotation-policy",
				Message:     "no rotation policy set (use: envs3 meta set KEY --rotation-days=90)",
			})
		}
	}

	// Check 5: Audit logging availability
	auditPrefix := e.Project + "/audit/"
	auditKeys, _ := e.Store.List(ctx, auditPrefix)
	if len(auditKeys) == 0 {
		issues = append(issues, ComplianceIssue{
			Severity:    "warning",
			Environment: "",
			Key:         "",
			Rule:        "no-audit-log",
			Message:     "no audit entries found — operations may not be logged",
		})
	}

	// Check 6: Token expiry
	var members format.MembersFile
	if _, err := e.getJSON(ctx, storage.MembersPath(e.Project), &members); err == nil {
		for _, token := range members.Tokens {
			if token.ExpiresAt == nil {
				issues = append(issues, ComplianceIssue{
					Severity:    "warning",
					Environment: token.Environment,
					Key:         "",
					Rule:        "token-no-expiry",
					Message:     fmt.Sprintf("token '%s' has no expiry set", token.Name),
				})
			} else {
				expiresAt, err := time.Parse(time.RFC3339, *token.ExpiresAt)
				if err == nil && now.After(expiresAt) {
					issues = append(issues, ComplianceIssue{
						Severity:    "critical",
						Environment: token.Environment,
						Key:         "",
						Rule:        "token-expired",
						Message:     fmt.Sprintf("token '%s' expired at %s", token.Name, *token.ExpiresAt),
					})
				}
			}
		}
	}

	return issues, nil
}

// SetSecretMetadata sets metadata for a specific secret key.
func (e *Engine) SetSecretMetadata(ctx context.Context, env, email string, key string, meta format.SecretMetadata, priv, pub [32]byte) error {
	var bundle format.Bundle
	currentETag, err := e.getJSON(ctx, storage.CurrentPath(e.Project, env), &bundle)
	if err != nil {
		return fmt.Errorf("fetch current: %w", err)
	}

	if _, exists := bundle.Secrets[key]; !exists {
		return fmt.Errorf("key '%s' not found in %s", key, env)
	}

	if bundle.Metadata == nil {
		bundle.Metadata = make(map[string]format.SecretMetadata)
	}

	// Merge with existing metadata
	existing := bundle.Metadata[key]
	if meta.RotatedAt != "" {
		existing.RotatedAt = meta.RotatedAt
	}
	if meta.RotationDays > 0 {
		existing.RotationDays = meta.RotationDays
	}
	if meta.ExpiresAt != "" {
		existing.ExpiresAt = meta.ExpiresAt
	}
	if len(meta.Tags) > 0 {
		existing.Tags = meta.Tags
	}
	if meta.Description != "" {
		existing.Description = meta.Description
	}
	bundle.Metadata[key] = existing

	bundle.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	bundle.UpdatedBy = email

	if _, err := e.putJSON(ctx, storage.CurrentPath(e.Project, env), &bundle, currentETag); err != nil {
		return fmt.Errorf("update metadata: %w", err)
	}

	return nil
}
