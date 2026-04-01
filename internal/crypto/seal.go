package crypto

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/nacl/box"
)

// SealBox encrypts plaintext using NaCl crypto_box_seal (anonymous sealed box).
// Only the recipient's public key is needed to encrypt.
// Output is 48 bytes larger than input (32-byte ephemeral public key + 16-byte MAC).
func SealBox(plaintext []byte, recipientPub [32]byte) ([]byte, error) {
	sealed, err := box.SealAnonymous(nil, plaintext, &recipientPub, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("seal: %w", err)
	}
	return sealed, nil
}

// OpenBox decrypts a NaCl sealed box using the recipient's keypair.
func OpenBox(sealed []byte, recipientPub, recipientPriv [32]byte) ([]byte, error) {
	plaintext, ok := box.OpenAnonymous(nil, sealed, &recipientPub, &recipientPriv)
	if !ok {
		return nil, fmt.Errorf("unseal: decryption failed (key not authorized)")
	}
	return plaintext, nil
}
