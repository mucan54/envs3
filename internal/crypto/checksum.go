package crypto

import (
	"crypto/sha256"
	"encoding/hex"
)

// ComputeChecksum returns "sha256:<hex>" of the given data.
func ComputeChecksum(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}
