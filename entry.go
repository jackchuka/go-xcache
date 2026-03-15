package xcache

import "time"

// entry wraps cached values with metadata. Struct tags are for the default JSON codec.
// Custom codecs see exported field names (Key, ExpiresAt, Value).
type entry[V any] struct {
	Key       string    `json:"key"`
	ExpiresAt time.Time `json:"expires_at"`
	Value     V         `json:"value"`
}

func (e entry[V]) isExpired() bool {
	return time.Now().After(e.ExpiresAt)
}
