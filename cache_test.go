package xcache

import (
	"testing"
	"time"

	"github.com/adrg/xdg"
)

type sample struct {
	Name string `json:"name"`
}

func testDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Set xdg.CacheHome directly since the library reads env at init time
	orig := xdg.CacheHome
	xdg.CacheHome = dir
	t.Cleanup(func() { xdg.CacheHome = orig })
	return dir
}

func TestNew_CreatesInstance(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool")
	if c == nil {
		t.Fatal("New returned nil")
	}
}

func TestNew_WithNamespace(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("ns"))
	if c == nil {
		t.Fatal("New returned nil")
	}
}

func TestNew_PanicsOnEmptyToolName(t *testing.T) {
	testDir(t)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty toolName")
		}
	}()
	New[sample]("")
}

func TestNew_PanicsOnUnsafeToolName(t *testing.T) {
	testDir(t)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unsafe toolName")
		}
	}()
	New[sample]("../escape")
}

func TestNew_PanicsOnUnsafeNamespace(t *testing.T) {
	testDir(t)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unsafe namespace")
		}
	}()
	New[sample]("tool", WithNamespace("../escape"))
}

func TestNew_PanicsOnNegativeDefaultTTL(t *testing.T) {
	testDir(t)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative default TTL")
		}
	}()
	New[sample]("tool", WithDefaultTTL(-1*time.Second))
}
