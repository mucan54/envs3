package engine

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mucan54/envs3/internal/format"
)

// AuditLog writes an audit entry to S3.
func (e *Engine) AuditLog(ctx context.Context, entry *format.AuditEntry) {
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	entry.SchemaVersion = 1

	// Path: <project>/audit/<date>/<timestamp>_<action>.json
	date := entry.Timestamp[:10] // YYYY-MM-DD
	ts := strings.ReplaceAll(entry.Timestamp, ":", "-")
	key := fmt.Sprintf("%s/audit/%s/%s_%s.json", e.Project, date, ts, entry.Action)

	// Best-effort — don't fail the operation if audit logging fails
	e.putJSON(ctx, key, entry, "")
}

// ListAuditLog retrieves audit entries, optionally filtered.
func (e *Engine) ListAuditLog(ctx context.Context, env, actor, action string, limit int) ([]format.AuditEntry, error) {
	prefix := e.Project + "/audit/"
	keys, err := e.Store.List(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("list audit log: %w", err)
	}

	// Sort reverse chronological
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	var entries []format.AuditEntry
	for _, key := range keys {
		if limit > 0 && len(entries) >= limit {
			break
		}

		var entry format.AuditEntry
		if _, err := e.getJSON(ctx, key, &entry); err != nil {
			continue
		}

		// Apply filters
		if env != "" && entry.Environment != env {
			continue
		}
		if actor != "" && entry.Actor != actor {
			continue
		}
		if action != "" && entry.Action != action {
			continue
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
