package xcache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
)

// Cache is a generic, file-based, TTL-aware cache.
type Cache[V any] struct {
	dir        string
	defaultTTL time.Duration
	codec      Codec
	keyFunc    KeyFunc
}

// New creates a cache instance. Panics if toolName, namespace, or options are invalid.
func New[V any](toolName string, opts ...Option) *Cache[V] {
	validateName(toolName, "toolName")

	cfg := config{
		codec:   JSONCodec{},
		keyFunc: SHA256KeyFunc,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.namespace != "" {
		validateName(cfg.namespace, "namespace")
	}
	if cfg.defaultTTL < 0 {
		panic("xcache: defaultTTL must not be negative")
	}

	dir := filepath.Join(xdg.CacheHome, toolName)
	if cfg.namespace != "" {
		dir = filepath.Join(dir, cfg.namespace)
	}

	return &Cache[V]{
		dir:        dir,
		defaultTTL: cfg.defaultTTL,
		codec:      cfg.codec,
		keyFunc:    cfg.keyFunc,
	}
}

func validateName(name, label string) {
	if name == "" {
		panic(fmt.Sprintf("xcache: %s must not be empty", label))
	}
	if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		panic(fmt.Sprintf("xcache: %s %q contains unsafe path components", label, name))
	}
}

func (c *Cache[V]) filepath(key string) string {
	return filepath.Join(c.dir, c.keyFunc(key)+c.codec.Extension())
}

// Get returns the cached value. Returns (zero, false) if missing, expired, or unreadable.
// Expired entries are lazily deleted from disk.
func (c *Cache[V]) Get(key string) (V, bool) {
	var zero V
	if key == "" {
		return zero, false
	}

	path := c.filepath(key)
	data, err := os.ReadFile(path)
	if err != nil {
		return zero, false
	}

	var e entry[V]
	if err := c.codec.Unmarshal(data, &e); err != nil {
		// Corrupt file — delete it
		os.Remove(path)
		return zero, false
	}

	if e.isExpired() {
		os.Remove(path)
		return zero, false
	}

	return e.Value, true
}

// Set stores a value with the given TTL. Uses default TTL if ttl is 0.
func (c *Cache[V]) Set(key string, value V, ttl time.Duration) error {
	if key == "" {
		return fmt.Errorf("xcache: key must not be empty")
	}

	if ttl == 0 {
		ttl = c.defaultTTL
	}
	if ttl <= 0 {
		return fmt.Errorf("xcache: ttl must be positive (got %v), set a default with WithDefaultTTL", ttl)
	}

	e := entry[V]{
		Key:       key,
		ExpiresAt: time.Now().Add(ttl),
		Value:     value,
	}

	data, err := c.codec.Marshal(e)
	if err != nil {
		return fmt.Errorf("xcache: marshal: %w", err)
	}

	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return fmt.Errorf("xcache: mkdir: %w", err)
	}

	// Atomic write: temp file + rename
	tmp, err := os.CreateTemp(c.dir, ".xcache-*")
	if err != nil {
		return fmt.Errorf("xcache: create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("xcache: write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("xcache: close: %w", err)
	}

	finalPath := c.filepath(key)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("xcache: rename: %w", err)
	}

	return nil
}

// Delete removes a single cache entry. Returns nil if the entry does not exist.
func (c *Cache[V]) Delete(key string) error {
	path := c.filepath(key)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("xcache: delete: %w", err)
	}
	return nil
}

// Clear removes all files in this cache's directory.
// Only removes files, not subdirectories (which may be other namespaces).
func (c *Cache[V]) Clear() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("xcache: clear: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(c.dir, e.Name())
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("xcache: clear: %w", err)
		}
	}
	return nil
}

// GetOrSet returns the cached value if present, otherwise calls fn to fetch it,
// caches the result, and returns it. If fn succeeds but Set fails, the value
// is still returned alongside the Set error.
func (c *Cache[V]) GetOrSet(key string, ttl time.Duration, fn func() (V, error)) (V, error) {
	if key == "" {
		var zero V
		return zero, fmt.Errorf("xcache: key must not be empty")
	}

	if v, ok := c.Get(key); ok {
		return v, nil
	}

	v, err := fn()
	if err != nil {
		var zero V
		return zero, err
	}

	setErr := c.Set(key, v, ttl)
	return v, setErr
}
