# go-xcache

[![Go Reference](https://pkg.go.dev/badge/github.com/jackchuka/go-xcache.svg)](https://pkg.go.dev/github.com/jackchuka/go-xcache)
[![Go Report Card](https://goreportcard.com/badge/github.com/jackchuka/go-xcache)](https://goreportcard.com/report/github.com/jackchuka/go-xcache)
[![Test](https://github.com/jackchuka/go-xcache/actions/workflows/test.yml/badge.svg)](https://github.com/jackchuka/go-xcache/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

A generic, file-based, TTL-aware cache for Go CLI tools. Stores any struct under `$XDG_CACHE_HOME` with per-entry TTL, pluggable serialization, and configurable key encoding.

## Install

```
go get github.com/jackchuka/go-xcache
```

## Usage

```go
import "github.com/jackchuka/go-xcache"

// Create a typed cache
releases := xcache.New[Release]("my-tool",
    xcache.WithNamespace("releases"),
    xcache.WithDefaultTTL(24 * time.Hour),
)

// Get / Set
releases.Set("owner/repo", release, 24*time.Hour)
val, ok := releases.Get("owner/repo")

// Fetch-through pattern
val, err := releases.GetOrSet("owner/repo", 0, func() (Release, error) {
    return client.GetLatestRelease(owner, repo)
})

// Cleanup
releases.Delete("owner/repo")
releases.Clear()
```

## API

| Method | Description |
|--------|-------------|
| `New[V](toolName, ...Option)` | Create a cache. Panics on invalid inputs. |
| `Get(key)` | Returns `(value, true)` on hit, `(zero, false)` on miss/expired/error. |
| `Set(key, value, ttl)` | Stores a value. Returns error on disk failure. |
| `GetOrSet(key, ttl, fn)` | Returns cached value or calls `fn`, caches result. |
| `Delete(key)` | Removes an entry. |
| `Clear()` | Removes all entries in this cache's namespace. |

## Options

```go
xcache.WithNamespace("releases")      // subdirectory under tool cache
xcache.WithDefaultTTL(time.Hour)      // fallback when Set gets ttl=0
xcache.WithCodec(myCodec)             // default: JSON
xcache.WithKeyFunc(xcache.SafeNameKeyFunc) // default: SHA-256 hash
```

## On-Disk Layout

```
~/.cache/
  my-tool/
    releases/
      a1b2c3d4e5f6...json    # SHA-256 of key
```

Each file:

```json
{
  "key": "owner/repo",
  "expires_at": "2026-03-15T14:30:00Z",
  "value": { "tag": "v1.2.3" }
}
```

## Key Functions

- **`SHA256KeyFunc`** (default) -- safe, fixed-length, not human-readable
- **`SafeNameKeyFunc`** -- replaces `/\:*?"<>|` with `_`, readable filenames. Falls back to SHA-256 for keys >200 chars.

## Custom Codec

Implement the `Codec` interface:

```go
type Codec interface {
    Marshal(v any) ([]byte, error)
    Unmarshal(data []byte, v any) error
    Extension() string // e.g. ".json", ".gob"
}
```

## Design

- **One file per key** -- no locking, safe for concurrent goroutines
- **Atomic writes** -- temp file + `os.Rename`
- **Lazy expiry** -- expired files deleted on `Get`, no background cleanup
- **XDG compliant** -- respects `$XDG_CACHE_HOME` via [adrg/xdg](https://github.com/adrg/xdg)
- **Get never fails** -- returns `(zero, false)` on any error
- **Set returns errors** -- caller decides whether to handle

## License

MIT
