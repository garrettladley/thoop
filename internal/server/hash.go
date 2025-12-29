package server

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashSecret returns the SHA256 hash of a secret string (e.g., API key, token).
// Used for secure storage and comparison without storing plaintext secrets.
func HashSecret(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
