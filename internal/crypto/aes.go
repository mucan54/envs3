package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateDEK generates a 32-byte random Data Encryption Key for AES-256.
func GenerateDEK() ([]byte, error) {
	dek := make([]byte, 32)
	if _, err := rand.Read(dek); err != nil {
		return nil, err
	}
	return dek, nil
}

// EncryptValue encrypts a plaintext value using AES-256-GCM.
// Returns base64-encoded ciphertext, nonce, and tag.
func EncryptValue(dek []byte, plaintext []byte) (value, nonce, tag string, err error) {
	block, err := aes.NewCipher(dek)
	if err != nil {
		return "", "", "", fmt.Errorf("aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", "", fmt.Errorf("gcm: %w", err)
	}

	nonceBytes := make([]byte, gcm.NonceSize()) // 12 bytes
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", "", "", fmt.Errorf("nonce: %w", err)
	}

	// GCM Seal appends the tag to the ciphertext
	sealed := gcm.Seal(nil, nonceBytes, plaintext, nil)

	// Split ciphertext and tag (tag is last 16 bytes)
	tagSize := gcm.Overhead() // 16 bytes
	ct := sealed[:len(sealed)-tagSize]
	tagBytes := sealed[len(sealed)-tagSize:]

	return base64.StdEncoding.EncodeToString(ct),
		base64.StdEncoding.EncodeToString(nonceBytes),
		base64.StdEncoding.EncodeToString(tagBytes),
		nil
}

// DecryptValue decrypts an AES-256-GCM encrypted value.
// Inputs are base64-encoded ciphertext, nonce, and tag.
func DecryptValue(dek []byte, valueB64, nonceB64, tagB64 string) ([]byte, error) {
	ct, err := base64.StdEncoding.DecodeString(valueB64)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}

	nonceBytes, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}

	tagBytes, err := base64.StdEncoding.DecodeString(tagB64)
	if err != nil {
		return nil, fmt.Errorf("decode tag: %w", err)
	}

	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	// Reassemble ciphertext + tag for GCM Open
	sealed := append(ct, tagBytes...)

	plaintext, err := gcm.Open(nil, nonceBytes, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: authentication failed: %w", err)
	}

	return plaintext, nil
}
