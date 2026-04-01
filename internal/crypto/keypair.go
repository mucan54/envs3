package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"

	"golang.org/x/crypto/curve25519"
)

// GenerateKeypair creates a new X25519 keypair.
func GenerateKeypair() (pub, priv [32]byte, err error) {
	if _, err = rand.Read(priv[:]); err != nil {
		return pub, priv, err
	}
	pubSlice, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return pub, priv, err
	}
	copy(pub[:], pubSlice)
	return pub, priv, nil
}

// PublicKeyFromPrivate derives the X25519 public key from a private key.
func PublicKeyFromPrivate(priv [32]byte) ([32]byte, error) {
	var pub [32]byte
	pubSlice, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return pub, err
	}
	copy(pub[:], pubSlice)
	return pub, nil
}

// Fingerprint computes the key fingerprint: "SHA256:" + first 12 chars of base64(SHA256(pubkey)).
func Fingerprint(pub [32]byte) string {
	h := sha256.Sum256(pub[:])
	encoded := base64.StdEncoding.EncodeToString(h[:])
	if len(encoded) > 12 {
		encoded = encoded[:12]
	}
	return "SHA256:" + encoded
}

// EncodeKey encodes a 32-byte key as standard base64 with padding.
func EncodeKey(key [32]byte) string {
	return base64.StdEncoding.EncodeToString(key[:])
}

// DecodeKey decodes a standard base64 string into a 32-byte key.
func DecodeKey(s string) ([32]byte, error) {
	var key [32]byte
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return key, err
	}
	copy(key[:], data)
	return key, nil
}
