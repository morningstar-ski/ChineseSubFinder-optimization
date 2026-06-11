package file_downloader

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// BuildCacheKey keeps provider cache identifiers stable without leaking raw URLs
// into filesystem paths on Windows.
func BuildCacheKey(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "::")))
	return hex.EncodeToString(sum[:])
}
