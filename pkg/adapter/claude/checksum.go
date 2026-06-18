package claude

import (
	"crypto/sha256"
	"encoding/hex"
)

// checksum returns the SHA256 checksum of s.
func checksum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
