package xcache

import (
	"fmt"
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
