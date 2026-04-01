package format

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseDotEnvBasic(t *testing.T) {
	input := `# comment
APP_NAME=TrailerTec
DB_HOST=localhost
DB_PORT=5432
`
	result, err := ParseDotEnv(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	if result["APP_NAME"] != "TrailerTec" {
		t.Errorf("APP_NAME = %q", result["APP_NAME"])
	}
	if result["DB_HOST"] != "localhost" {
		t.Errorf("DB_HOST = %q", result["DB_HOST"])
	}
	if result["DB_PORT"] != "5432" {
		t.Errorf("DB_PORT = %q", result["DB_PORT"])
	}
	if len(result) != 3 {
		t.Errorf("expected 3 keys, got %d", len(result))
	}
}

func TestParseDotEnvQuoted(t *testing.T) {
	input := `KEY1="hello world"
KEY2='literal value'
KEY3="with \"escape\""
KEY4="with\nnewline"
`
	result, err := ParseDotEnv(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	if result["KEY1"] != "hello world" {
		t.Errorf("KEY1 = %q", result["KEY1"])
	}
	if result["KEY2"] != "literal value" {
		t.Errorf("KEY2 = %q", result["KEY2"])
	}
	if result["KEY3"] != `with "escape"` {
		t.Errorf("KEY3 = %q", result["KEY3"])
	}
	if result["KEY4"] != "with\nnewline" {
		t.Errorf("KEY4 = %q", result["KEY4"])
	}
}

func TestParseDotEnvInlineComment(t *testing.T) {
	input := `KEY=value # comment`
	result, err := ParseDotEnv(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY"] != "value" {
		t.Errorf("KEY = %q", result["KEY"])
	}
}

func TestParseDotEnvEmptyValue(t *testing.T) {
	input := `KEY=`
	result, err := ParseDotEnv(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY"] != "" {
		t.Errorf("KEY = %q", result["KEY"])
	}
}

func TestWriteDotEnv(t *testing.T) {
	secrets := map[string]string{
		"DB_PORT":  "5432",
		"APP_NAME": "test",
		"DB_HOST":  "localhost",
	}

	var buf bytes.Buffer
	WriteDotEnv(&buf, secrets, "local", 1, "2026-04-01T10:00:00Z")

	output := buf.String()
	if !strings.Contains(output, "# envs3 |") {
		t.Error("missing header")
	}

	// Keys should be sorted
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var kvLines []string
	for _, l := range lines {
		if !strings.HasPrefix(l, "#") && l != "" {
			kvLines = append(kvLines, l)
		}
	}
	if len(kvLines) != 3 {
		t.Fatalf("expected 3 kv lines, got %d", len(kvLines))
	}
	if !strings.HasPrefix(kvLines[0], "APP_NAME=") {
		t.Errorf("first key should be APP_NAME, got %s", kvLines[0])
	}
}

func TestWriteDotEnvQuoting(t *testing.T) {
	secrets := map[string]string{
		"NORMAL": "value",
		"SPACES": "has spaces",
		"HASH":   "has#hash",
	}

	var buf bytes.Buffer
	WriteDotEnv(&buf, secrets, "local", 1, "2026-04-01T10:00:00Z")

	output := buf.String()
	if !strings.Contains(output, `SPACES="has spaces"`) {
		t.Errorf("SPACES not properly quoted in: %s", output)
	}
	if !strings.Contains(output, `HASH="has#hash"`) {
		t.Errorf("HASH not properly quoted in: %s", output)
	}
	if strings.Contains(output, `NORMAL="value"`) {
		t.Error("NORMAL should not be quoted")
	}
}

func TestDotEnvRoundTrip(t *testing.T) {
	secrets := map[string]string{
		"KEY1": "simple",
		"KEY2": "with spaces",
		"KEY3": "with\"quotes",
	}

	var buf bytes.Buffer
	WriteDotEnv(&buf, secrets, "test", 1, "2026-04-01T10:00:00Z")

	parsed, err := ParseDotEnv(&buf)
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range secrets {
		if parsed[k] != v {
			t.Errorf("%s: got %q, want %q", k, parsed[k], v)
		}
	}
}

func TestCanonicalJSON(t *testing.T) {
	secrets := map[string]EncryptedSecret{
		"B_KEY": {Value: "v2", Nonce: "n2", Tag: "t2"},
		"A_KEY": {Value: "v1", Nonce: "n1", Tag: "t1"},
	}

	j1, err := CanonicalJSON(secrets)
	if err != nil {
		t.Fatal(err)
	}

	// Should be deterministic
	j2, err := CanonicalJSON(secrets)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(j1, j2) {
		t.Fatal("canonical JSON not deterministic")
	}

	// Keys should be sorted
	if !strings.Contains(string(j1), `"A_KEY"`) {
		t.Error("missing A_KEY")
	}
	if strings.Index(string(j1), `"A_KEY"`) > strings.Index(string(j1), `"B_KEY"`) {
		t.Error("keys not sorted: A_KEY should come before B_KEY")
	}

	// Should be valid JSON
	var m map[string]interface{}
	if err := json.Unmarshal(j1, &m); err != nil {
		t.Fatalf("invalid JSON: %s", err)
	}
}

func TestBundleJSONRoundTrip(t *testing.T) {
	bundle := Bundle{
		SchemaVersion: 1,
		Environment:   "local",
		Version:       3,
		DEKID:         "dek_local_v1",
		UpdatedAt:     "2026-04-01T14:30:00Z",
		UpdatedBy:     "test@test.com",
		Checksum:      "sha256:abc123",
		Secrets: map[string]EncryptedSecret{
			"KEY1": {Value: "v1", Nonce: "n1", Tag: "t1"},
		},
	}

	data, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Bundle
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Version != 3 {
		t.Errorf("version: got %d, want 3", decoded.Version)
	}
	if decoded.DEKID != "dek_local_v1" {
		t.Errorf("dek_id: got %s", decoded.DEKID)
	}
	if len(decoded.Secrets) != 1 {
		t.Errorf("secrets: got %d, want 1", len(decoded.Secrets))
	}
}

func TestKeyringJSONRoundTrip(t *testing.T) {
	keyring := KeyringFile{
		SchemaVersion: 1,
		DEKID:         "dek_local_v1",
		Algorithm:     "x25519-xsalsa20-poly1305",
		CreatedAt:     "2026-04-01T10:00:00Z",
		Wrapped: []WrappedEntry{
			{Fingerprint: "SHA256:abc123", WrappedDEK: "base64data"},
		},
	}

	data, err := json.Marshal(keyring)
	if err != nil {
		t.Fatal(err)
	}

	var decoded KeyringFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.DEKID != "dek_local_v1" {
		t.Errorf("dek_id: got %s", decoded.DEKID)
	}
	if len(decoded.Wrapped) != 1 {
		t.Errorf("wrapped: got %d, want 1", len(decoded.Wrapped))
	}
}
