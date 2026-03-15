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
