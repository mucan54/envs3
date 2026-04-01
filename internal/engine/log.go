package engine

import (
	"context"
	"fmt"
	"sort"

	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
)

// LogEntry is a summary of a history version.
type LogEntry struct {
	Version   int
	CreatedAt string
	CreatedBy string
	Changes   []format.Change
}

// Log returns version history for an environment.
func (e *Engine) Log(ctx context.Context, env string, limit int) ([]LogEntry, error) {
	prefix := storage.HistoryPrefix(e.Project, env)
	keys, err := e.Store.List(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	if limit > 0 && len(keys) > limit {
		keys = keys[:limit]
	}

	var entries []LogEntry
	for _, key := range keys {
		var h format.HistoryEntry
		if _, err := e.getJSON(ctx, key, &h); err != nil {
			continue
		}
		entries = append(entries, LogEntry{
			Version:   h.Version,
			CreatedAt: h.CreatedAt,
			CreatedBy: h.CreatedBy,
			Changes:   h.Changes,
		})
	}

	return entries, nil
}
