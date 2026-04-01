package storage

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type memEntry struct {
	data []byte
	etag string
}

// MemoryStore is an in-memory implementation of Store for testing.
type MemoryStore struct {
	mu      sync.RWMutex
	objects map[string]memEntry
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		objects: make(map[string]memEntry),
	}
}

func computeETag(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf(`"%x"`, h[:8])
}

func (m *MemoryStore) Get(_ context.Context, key string) ([]byte, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.objects[key]
	if !ok {
		return nil, "", ErrNotFound
	}
	// Return a copy to prevent mutation
	cp := make([]byte, len(entry.data))
	copy(cp, entry.data)
	return cp, entry.etag, nil
}

func (m *MemoryStore) Put(_ context.Context, key string, data []byte, ifMatchETag string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ifMatchETag != "" {
		existing, ok := m.objects[key]
		if !ok {
			return "", ErrConflict
		}
		if existing.etag != ifMatchETag {
			return "", ErrConflict
		}
	}

	cp := make([]byte, len(data))
	copy(cp, data)
	etag := computeETag(data)
	m.objects[key] = memEntry{data: cp, etag: etag}
	return etag, nil
}

func (m *MemoryStore) Head(_ context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.objects[key]
	if !ok {
		return "", ErrNotFound
	}
	return entry.etag, nil
}

func (m *MemoryStore) List(_ context.Context, prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var keys []string
	for k := range m.objects {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func (m *MemoryStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.objects, key)
	return nil
}
