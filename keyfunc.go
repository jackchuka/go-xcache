package xcache

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// KeyFunc transforms a cache key into a filename-safe string.
type KeyFunc func(key string) string

// SHA256KeyFunc hashes the key with SHA-256, producing a 64-char hex string.
func SHA256KeyFunc(key string) string {
	h := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", h)
}

// SafeNameKeyFunc replaces filename-unsafe characters with underscores.
// Falls back to SHA256KeyFunc for keys longer than 200 characters.
func SafeNameKeyFunc(key string) string {
	if len(key) > 200 {
		return SHA256KeyFunc(key)
	}
	replacer := strings.NewReplacer(
		"/", "_",
		`\`, "_",
		":", "_",
		"*", "_",
		"?", "_",
		`"`, "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(key)
}
