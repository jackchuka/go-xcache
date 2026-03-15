package xcache

import (
	"os"
	"path/filepath"
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

func TestSet_And_Get(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("data"))

	err := c.Set("k1", sample{Name: "hello"}, time.Hour)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, ok := c.Get("k1")
	if !ok {
		t.Fatal("Get returned false for existing key")
	}
	if val.Name != "hello" {
		t.Errorf("Name = %q, want %q", val.Name, "hello")
	}
}

func TestGet_Missing(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("data"))

	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("Get returned true for missing key")
	}
}

func TestGet_Expired(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("data"))

	_ = c.Set("k1", sample{Name: "old"}, time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get("k1")
	if ok {
		t.Error("Get returned true for expired key")
	}
}

func TestGet_ExpiredDeletesFile(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("data"))

	_ = c.Set("k1", sample{Name: "old"}, time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	c.Get("k1")

	path := c.filepath("k1")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expired file was not deleted on Get")
	}
}

func TestGet_CorruptFile(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("data"))

	path := c.filepath("k1")
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte("not json"), 0o644)

	_, ok := c.Get("k1")
	if ok {
		t.Error("Get returned true for corrupt file")
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("corrupt file was not deleted")
	}
}

func TestSet_EmptyKey(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool")

	err := c.Set("", sample{}, time.Hour)
	if err == nil {
		t.Error("Set with empty key should return error")
	}
}

func TestGet_EmptyKey(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool")

	_, ok := c.Get("")
	if ok {
		t.Error("Get with empty key should return false")
	}
}

func TestSet_ZeroTTLNoDefault(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool")

	err := c.Set("k1", sample{}, 0)
	if err == nil {
		t.Error("Set with zero TTL and no default should return error")
	}
}

func TestSet_ZeroTTLUsesDefault(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithDefaultTTL(time.Hour))

	err := c.Set("k1", sample{Name: "default-ttl"}, 0)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, ok := c.Get("k1")
	if !ok {
		t.Fatal("Get returned false")
	}
	if val.Name != "default-ttl" {
		t.Errorf("Name = %q, want %q", val.Name, "default-ttl")
	}
}

func TestSet_AtomicWrite(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("atomic"))

	_ = c.Set("k1", sample{Name: "first"}, time.Hour)
	_ = c.Set("k1", sample{Name: "second"}, time.Hour)

	val, ok := c.Get("k1")
	if !ok {
		t.Fatal("Get returned false")
	}
	if val.Name != "second" {
		t.Errorf("Name = %q, want %q", val.Name, "second")
	}
}
