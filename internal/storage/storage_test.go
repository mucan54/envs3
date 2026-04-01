package storage

import (
	"context"
	"testing"
)

func TestMemoryStoreBasic(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Put
	etag, err := store.Put(ctx, "test/key", []byte("hello"), "")
	if err != nil {
		t.Fatal(err)
	}
	if etag == "" {
		t.Fatal("empty etag")
	}

	// Get
	data, getEtag, err := store.Get(ctx, "test/key")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("got %q, want %q", data, "hello")
	}
	if getEtag != etag {
		t.Fatal("etag mismatch")
	}

	// Head
	headEtag, err := store.Head(ctx, "test/key")
	if err != nil {
		t.Fatal(err)
	}
	if headEtag != etag {
		t.Fatal("head etag mismatch")
	}
}

func TestMemoryStoreNotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	_, _, err := store.Get(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	_, err = store.Head(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStoreConditionalPut(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Initial put
	etag1, _ := store.Put(ctx, "key", []byte("v1"), "")

	// Conditional put with correct etag
	etag2, err := store.Put(ctx, "key", []byte("v2"), etag1)
	if err != nil {
		t.Fatal(err)
	}
	if etag2 == etag1 {
		t.Fatal("etag should change")
	}

	// Conditional put with stale etag should fail
	_, err = store.Put(ctx, "key", []byte("v3"), etag1)
	if err != ErrConflict {
		t.Fatalf("expected ErrConflict, got %v", err)
	}

	// Conditional put to nonexistent key should fail
	_, err = store.Put(ctx, "newkey", []byte("data"), "some-etag")
	if err != ErrConflict {
		t.Fatalf("expected ErrConflict for nonexistent key, got %v", err)
	}
}

func TestMemoryStoreList(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	store.Put(ctx, "project/env/a", []byte("1"), "")
	store.Put(ctx, "project/env/b", []byte("2"), "")
	store.Put(ctx, "project/other/c", []byte("3"), "")

	keys, err := store.List(ctx, "project/env/")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestMemoryStoreDelete(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	store.Put(ctx, "key", []byte("data"), "")

	err := store.Delete(ctx, "key")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = store.Get(ctx, "key")
	if err != ErrNotFound {
		t.Fatal("expected ErrNotFound after delete")
	}
}

func TestMemoryStoreDataIsolation(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	original := []byte("original")
	store.Put(ctx, "key", original, "")

	// Modify original slice
	original[0] = 'X'

	data, _, _ := store.Get(ctx, "key")
	if data[0] == 'X' {
		t.Fatal("store should copy data, not reference it")
	}
}

func TestPathFunctions(t *testing.T) {
	tests := []struct {
		name string
		fn   func() string
		want string
	}{
		{"ProjectPath", func() string { return ProjectPath("myproj") }, "myproj/project.json"},
		{"MembersPath", func() string { return MembersPath("myproj") }, "myproj/members.json"},
		{"KeyringPath", func() string { return KeyringPath("myproj", "local") }, "myproj/environments/local/keyring.json"},
		{"CurrentPath", func() string { return CurrentPath("myproj", "local") }, "myproj/environments/local/current.json"},
		{"HistoryPath", func() string { return HistoryPath("myproj", "local", 42) }, "myproj/environments/local/history/000042.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
