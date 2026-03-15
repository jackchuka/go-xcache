package xcache

import "time"

type config struct {
	namespace  string
	defaultTTL time.Duration
	codec      Codec
	keyFunc    KeyFunc
}

// Option configures a Cache instance.
type Option func(*config)

// WithNamespace sets the subdirectory within the tool's cache directory.
func WithNamespace(ns string) Option {
	return func(c *config) {
		c.namespace = ns
	}
}

// WithDefaultTTL sets the fallback TTL used when Set is called with ttl=0.
func WithDefaultTTL(d time.Duration) Option {
	return func(c *config) {
		c.defaultTTL = d
	}
}

// WithCodec sets the serialization codec. Default is JSONCodec.
func WithCodec(codec Codec) Option {
	return func(c *config) {
		c.codec = codec
	}
}

// WithKeyFunc sets the key-to-filename function. Default is SHA256KeyFunc.
func WithKeyFunc(fn KeyFunc) Option {
	return func(c *config) {
		c.keyFunc = fn
	}
}
