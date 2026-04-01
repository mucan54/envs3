package crypto

import (
	"testing"
)

func TestGenerateKeypair(t *testing.T) {
	pub, priv, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}
	if pub == [32]byte{} {
		t.Fatal("public key is zero")
	}
	if priv == [32]byte{} {
		t.Fatal("private key is zero")
	}

	// Derive public key from private and verify
	derivedPub, err := PublicKeyFromPrivate(priv)
	if err != nil {
		t.Fatal(err)
	}
	if derivedPub != pub {
		t.Fatal("derived public key does not match")
	}
}

func TestFingerprint(t *testing.T) {
	pub, _, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	fp := Fingerprint(pub)
	if len(fp) < 10 {
		t.Fatalf("fingerprint too short: %s", fp)
	}
	if fp[:7] != "SHA256:" {
		t.Fatalf("fingerprint missing prefix: %s", fp)
	}

	// Same key should produce same fingerprint
	fp2 := Fingerprint(pub)
	if fp != fp2 {
		t.Fatal("fingerprints don't match for same key")
	}
}

func TestEncodeDecodeKey(t *testing.T) {
	pub, _, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	encoded := EncodeKey(pub)
	decoded, err := DecodeKey(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != pub {
		t.Fatal("round-trip failed")
	}
}

func TestSealUnseal(t *testing.T) {
	pub, priv, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("this is a secret DEK")
	sealed, err := SealBox(plaintext, pub)
	if err != nil {
		t.Fatal(err)
	}

	// Sealed should be 48 bytes larger than plaintext
	if len(sealed) != len(plaintext)+48 {
		t.Fatalf("unexpected sealed length: got %d, want %d", len(sealed), len(plaintext)+48)
	}

	// Unseal with correct key
	recovered, err := OpenBox(sealed, pub, priv)
	if err != nil {
		t.Fatal(err)
	}
	if string(recovered) != string(plaintext) {
		t.Fatal("decrypted text does not match")
	}

	// Unseal with wrong key should fail
	otherPub, otherPriv, _ := GenerateKeypair()
	_, err = OpenBox(sealed, otherPub, otherPriv)
	if err == nil {
		t.Fatal("expected error with wrong key")
	}
}

func TestAESEncryptDecrypt(t *testing.T) {
	dek, err := GenerateDEK()
	if err != nil {
		t.Fatal(err)
	}
	if len(dek) != 32 {
		t.Fatalf("DEK length: got %d, want 32", len(dek))
	}

	plaintext := []byte("my-secret-value-123!")
	value, nonce, tag, err := EncryptValue(dek, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	if value == "" || nonce == "" || tag == "" {
		t.Fatal("empty encryption output")
	}

	// Decrypt
	recovered, err := DecryptValue(dek, value, nonce, tag)
	if err != nil {
		t.Fatal(err)
	}
	if string(recovered) != string(plaintext) {
		t.Fatalf("decrypted %q, want %q", recovered, plaintext)
	}

	// Wrong DEK should fail
	wrongDEK, _ := GenerateDEK()
	_, err = DecryptValue(wrongDEK, value, nonce, tag)
	if err == nil {
		t.Fatal("expected error with wrong DEK")
	}
}

func TestAESNonceUniqueness(t *testing.T) {
	dek, _ := GenerateDEK()
	plaintext := []byte("test")

	_, nonce1, _, _ := EncryptValue(dek, plaintext)
	_, nonce2, _, _ := EncryptValue(dek, plaintext)

	if nonce1 == nonce2 {
		t.Fatal("nonces should be unique")
	}
}

func TestAESTamperDetection(t *testing.T) {
	dek, _ := GenerateDEK()
	plaintext := []byte("test-value")

	value, nonce, _, err := EncryptValue(dek, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	// Tamper with the tag
	tamperedTag := "AAAAAAAAAAAAAAAAAAAAAA=="
	_, err = DecryptValue(dek, value, nonce, tamperedTag)
	if err == nil {
		t.Fatal("expected error with tampered tag")
	}
}

func TestChecksum(t *testing.T) {
	data := []byte(`{"key":"value"}`)
	cs1 := ComputeChecksum(data)
	cs2 := ComputeChecksum(data)

	if cs1 != cs2 {
		t.Fatal("checksum not deterministic")
	}

	if cs1[:7] != "sha256:" {
		t.Fatalf("unexpected prefix: %s", cs1)
	}

	// Different data should produce different checksum
	cs3 := ComputeChecksum([]byte(`{"key":"other"}`))
	if cs1 == cs3 {
		t.Fatal("different data produced same checksum")
	}
}
