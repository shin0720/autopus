package claude

import (
	"crypto/sha256"
	"encoding/hex"
)

// checksum은 문자열의 SHA256 체크섬을 반환한다.
func checksum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
