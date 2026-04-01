package engine

import (
	"context"
	"testing"

	"github.com/mucan54/envs3/internal/crypto"
	"github.com/mucan54/envs3/internal/storage"
)

func setupTest(t *testing.T) (*Engine, [32]byte, [32]byte) {
	t.Helper()
	store := storage.NewMemoryStore()
	eng := NewEngine(store, "testproject")

	pub, priv, err := crypto.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	// Initialize project
	err = eng.Init(context.Background(), InitParams{
		ProjectName:  "testproject",
		Email:        "admin@test.com",
		Environments: []string{"local", "staging"},
		PublicKey:    pub,
	})
	if err != nil {
		t.Fatal(err)
	}

	return eng, priv, pub
}

func TestInitAndPull(t *testing.T) {
	eng, priv, pub := setupTest(t)
	ctx := context.Background()

	// Pull should work (empty env)
	result, err := eng.Pull(ctx, "local", priv, pub, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != 1 {
		t.Errorf("version: got %d, want 1", result.Version)
	}
	if len(result.Secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(result.Secrets))
	}
}

func TestInitWithImport(t *testing.T) {
	store := storage.NewMemoryStore()
	eng := NewEngine(store, "importproject")

	pub, priv, err := crypto.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	importData := map[string]string{
		"APP_NAME":    "TestApp",
		"DB_HOST":     "localhost",
		"DB_PASSWORD": "secret123",
	}

	err = eng.Init(context.Background(), InitParams{
		ProjectName:  "importproject",
		Email:        "admin@test.com",
		Environments: []string{"local"},
		ImportEnv:    "local",
		ImportData:   importData,
		PublicKey:    pub,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Pull and verify
	result, err := eng.Pull(context.Background(), "local", priv, pub, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Secrets) != 3 {
		t.Fatalf("expected 3 secrets, got %d", len(result.Secrets))
	}
	if result.Secrets["APP_NAME"] != "TestApp" {
		t.Errorf("APP_NAME: got %q", result.Secrets["APP_NAME"])
	}
	if result.Secrets["DB_PASSWORD"] != "secret123" {
		t.Errorf("DB_PASSWORD: got %q", result.Secrets["DB_PASSWORD"])
	}
}

func TestPushAndPull(t *testing.T) {
	eng, priv, pub := setupTest(t)
	ctx := context.Background()

	// Push some secrets
	secrets := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	}
	result, err := eng.Push(ctx, "local", "admin@test.com", secrets, priv, pub)
	if err != nil {
		t.Fatal(err)
	}
	if result.NewVersion != 2 {
		t.Errorf("version: got %d, want 2", result.NewVersion)
	}

	// Pull back
	pullResult, err := eng.Pull(ctx, "local", priv, pub, "")
	if err != nil {
		t.Fatal(err)
	}
	if pullResult.Secrets["KEY1"] != "value1" {
		t.Errorf("KEY1: got %q", pullResult.Secrets["KEY1"])
	}
	if pullResult.Secrets["KEY2"] != "value2" {
		t.Errorf("KEY2: got %q", pullResult.Secrets["KEY2"])
	}
}

func TestPullETagCache(t *testing.T) {
	eng, priv, pub := setupTest(t)
	ctx := context.Background()

	// First pull to get ETag
	result1, err := eng.Pull(ctx, "local", priv, pub, "")
	if err != nil {
		t.Fatal(err)
	}

	// Second pull with cached ETag should be up-to-date
	result2, err := eng.Pull(ctx, "local", priv, pub, result1.ETag)
	if err != nil {
		t.Fatal(err)
	}
	if !result2.UpToDate {
		t.Error("expected up-to-date")
	}
}

func TestSetRemoteOnly(t *testing.T) {
	eng, priv, pub := setupTest(t)
	ctx := context.Background()

	// Set a key remotely
	setResult, err := eng.Set(ctx, "local", "admin@test.com",
		map[string]string{"REMOTE_KEY": "remote_value"},
		nil, priv, pub)
	if err != nil {
		t.Fatal(err)
	}
	if setResult.NewVersion != 2 {
		t.Errorf("version: got %d, want 2", setResult.NewVersion)
	}

	// View to verify
	viewResult, err := eng.View(ctx, "local", priv, pub)
	if err != nil {
		t.Fatal(err)
	}
	if viewResult.Secrets["REMOTE_KEY"] != "remote_value" {
		t.Errorf("REMOTE_KEY: got %q", viewResult.Secrets["REMOTE_KEY"])
	}
}

func TestSetAndUnset(t *testing.T) {
	eng, priv, pub := setupTest(t)
	ctx := context.Background()

	// Add keys
	eng.Set(ctx, "local", "admin@test.com",
		map[string]string{"K1": "v1", "K2": "v2", "K3": "v3"},
		nil, priv, pub)

	// Update one, remove one
	result, err := eng.Set(ctx, "local", "admin@test.com",
		map[string]string{"K1": "updated"},
		[]string{"K3"}, priv, pub)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(result.Changes))
	}

	// Verify
	view, _ := eng.View(ctx, "local", priv, pub)
	if view.Secrets["K1"] != "updated" {
		t.Error("K1 not updated")
	}
	if _, exists := view.Secrets["K3"]; exists {
		t.Error("K3 should be removed")
	}
	if view.Secrets["K2"] != "v2" {
		t.Error("K2 should be unchanged")
	}
}

func TestViewKeys(t *testing.T) {
	eng, priv, pub := setupTest(t)
	ctx := context.Background()

	eng.Set(ctx, "local", "admin@test.com",
		map[string]string{"B_KEY": "b", "A_KEY": "a"},
		nil, priv, pub)

	keys, version, err := eng.ViewKeys(ctx, "local")
	if err != nil {
		t.Fatal(err)
	}
	if version != 2 {
		t.Errorf("version: got %d", version)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	// Keys should be sorted
	if keys[0] != "A_KEY" {
		t.Errorf("first key: got %s", keys[0])
	}
}

func TestDiff(t *testing.T) {
	eng, priv, pub := setupTest(t)
	ctx := context.Background()

	eng.Set(ctx, "local", "admin@test.com",
		map[string]string{"SHARED": "same", "LOCAL_ONLY": "val"},
		nil, priv, pub)

	eng.Set(ctx, "staging", "admin@test.com",
		map[string]string{"SHARED": "same", "STAGING_ONLY": "val"},
		nil, priv, pub)

	entries, err := eng.Diff(ctx, "local", "staging", priv, pub)
	if err != nil {
		t.Fatal(err)
	}

	found := make(map[string]string)
	for _, e := range entries {
		found[e.Key] = e.Status
	}

	if found["SHARED"] != "identical" {
		t.Errorf("SHARED: got %s", found["SHARED"])
	}
	if found["LOCAL_ONLY"] != "left_only" {
		t.Errorf("LOCAL_ONLY: got %s", found["LOCAL_ONLY"])
	}
	if found["STAGING_ONLY"] != "right_only" {
		t.Errorf("STAGING_ONLY: got %s", found["STAGING_ONLY"])
	}
}

func TestLog(t *testing.T) {
	eng, priv, pub := setupTest(t)
	ctx := context.Background()

	eng.Push(ctx, "local", "admin@test.com",
		map[string]string{"K1": "v1"}, priv, pub)
	eng.Push(ctx, "local", "admin@test.com",
		map[string]string{"K1": "v1", "K2": "v2"}, priv, pub)

	entries, err := eng.Log(ctx, "local", 10)
	if err != nil {
		t.Fatal(err)
	}

	// Should have history entries from pushes
	if len(entries) < 1 {
		t.Fatalf("expected at least 1 history entry, got %d", len(entries))
	}
}

func TestMemberAddAndAccess(t *testing.T) {
	eng, priv, pub := setupTest(t)
	ctx := context.Background()

	// Add a secret
	eng.Set(ctx, "local", "admin@test.com",
		map[string]string{"SECRET": "value"}, nil, priv, pub)

	// Create a new member keypair
	newPub, newPriv, err := crypto.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	// Add member with access to local
	err = eng.AddMember(ctx, newPub, "member@test.com", "member", []string{"local"}, priv, pub)
	if err != nil {
		t.Fatal(err)
	}

	// New member should be able to pull local
	result, err := eng.Pull(ctx, "local", newPriv, newPub, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Secrets["SECRET"] != "value" {
		t.Errorf("SECRET: got %q", result.Secrets["SECRET"])
	}

	// New member should NOT be able to pull staging
	_, err = eng.Pull(ctx, "staging", newPriv, newPub, "")
	if err == nil {
		t.Error("expected access denied for staging")
	}
}

func TestMemberRemoveWithDEKRotation(t *testing.T) {
	eng, priv, pub := setupTest(t)
	ctx := context.Background()

	eng.Set(ctx, "local", "admin@test.com",
		map[string]string{"SECRET": "value"}, nil, priv, pub)

	// Add member
	newPub, newPriv, _ := crypto.GenerateKeypair()
	eng.AddMember(ctx, newPub, "member@test.com", "member", []string{"local"}, priv, pub)

	// Verify member can access
	_, err := eng.Pull(ctx, "local", newPriv, newPub, "")
	if err != nil {
		t.Fatal("member should be able to pull before removal")
	}

	// Remove member
	err = eng.RemoveMember(ctx, "member@test.com", []string{"local"}, priv, pub, "admin@test.com")
	if err != nil {
		t.Fatal(err)
	}

	// Admin should still be able to pull (with rotated DEK)
	result, err := eng.Pull(ctx, "local", priv, pub, "")
	if err != nil {
		t.Fatalf("admin pull after rotation: %v", err)
	}
	if result.Secrets["SECRET"] != "value" {
		t.Error("secret value changed after rotation")
	}

	// Removed member should NOT be able to pull
	_, err = eng.Pull(ctx, "local", newPriv, newPub, "")
	if err == nil {
		t.Error("removed member should not be able to pull")
	}
}

func TestEnvCreate(t *testing.T) {
	eng, priv, pub := setupTest(t)
	ctx := context.Background()

	// Add secrets to local
	eng.Set(ctx, "local", "admin@test.com",
		map[string]string{"KEY": "value"}, nil, priv, pub)

	// Create new env from local
	err := eng.CreateEnv(ctx, "production", "admin@test.com", "local", priv, pub)
	if err != nil {
		t.Fatal(err)
	}

	// Pull from production
	result, err := eng.Pull(ctx, "production", priv, pub, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Secrets["KEY"] != "value" {
		t.Errorf("KEY: got %q", result.Secrets["KEY"])
	}
}

func TestChecksumVerification(t *testing.T) {
	eng, priv, pub := setupTest(t)
	ctx := context.Background()

	eng.Set(ctx, "local", "admin@test.com",
		map[string]string{"KEY": "value"}, nil, priv, pub)

	// Pull should verify checksum
	result, err := eng.Pull(ctx, "local", priv, pub, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Secrets["KEY"] != "value" {
		t.Error("checksum verification failed silently")
	}
}

func TestRollback(t *testing.T) {
	eng, priv, pub := setupTest(t)
	ctx := context.Background()

	// Push v2
	eng.Push(ctx, "local", "admin@test.com",
		map[string]string{"KEY": "v2value"}, priv, pub)

	// Push v3
	eng.Push(ctx, "local", "admin@test.com",
		map[string]string{"KEY": "v3value", "NEW": "new"}, priv, pub)

	// Rollback to v2
	result, err := eng.Rollback(ctx, "local", "admin@test.com", 2, priv, pub)
	if err != nil {
		t.Fatal(err)
	}
	if result.NewVersion != 4 {
		t.Errorf("version: got %d, want 4", result.NewVersion)
	}

	// Verify
	pullResult, _ := eng.Pull(ctx, "local", priv, pub, "")
	if pullResult.Secrets["KEY"] != "v2value" {
		t.Errorf("KEY: got %q, want v2value", pullResult.Secrets["KEY"])
	}
	if _, exists := pullResult.Secrets["NEW"]; exists {
		t.Error("NEW should not exist after rollback to v2")
	}
}

func TestListMembers(t *testing.T) {
	eng, priv, pub := setupTest(t)
	ctx := context.Background()

	newPub, _, _ := crypto.GenerateKeypair()
	eng.AddMember(ctx, newPub, "member@test.com", "member", []string{"local"}, priv, pub)

	members, err := eng.ListMembers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
}

func TestListEnvs(t *testing.T) {
	eng, _, _ := setupTest(t)
	ctx := context.Background()

	envs, err := eng.ListEnvs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(envs) != 2 {
		t.Fatalf("expected 2 envs, got %d", len(envs))
	}
}
