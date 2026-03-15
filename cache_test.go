package xcache

import (
	"fmt"
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

func TestDelete_Existing(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("data"))

	_ = c.Set("k1", sample{Name: "hello"}, time.Hour)

	if err := c.Delete("k1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, ok := c.Get("k1")
	if ok {
		t.Error("Get returned true after Delete")
	}
}

func TestDelete_NotFound(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool")

	err := c.Delete("nonexistent")
	if err != nil {
		t.Errorf("Delete nonexistent should not error, got: %v", err)
	}
}

func TestClear(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("clearme"))

	_ = c.Set("k1", sample{Name: "a"}, time.Hour)
	_ = c.Set("k2", sample{Name: "b"}, time.Hour)

	if err := c.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	_, ok1 := c.Get("k1")
	_, ok2 := c.Get("k2")
	if ok1 || ok2 {
		t.Error("entries remain after Clear")
	}
}

func TestClear_DoesNotRemoveOtherNamespaces(t *testing.T) {
	dir := testDir(t)
	c1 := New[sample]("test-tool", WithNamespace("ns1"))
	c2 := New[sample]("test-tool", WithNamespace("ns2"))

	_ = c1.Set("k1", sample{Name: "a"}, time.Hour)
	_ = c2.Set("k2", sample{Name: "b"}, time.Hour)

	_ = c1.Clear()

	val, ok := c2.Get("k2")
	if !ok {
		t.Error("Clear of ns1 removed ns2 entry")
	}
	if val.Name != "b" {
		t.Errorf("Name = %q, want %q", val.Name, "b")
	}

	ns2Dir := filepath.Join(dir, "test-tool", "ns2")
	if _, err := os.Stat(ns2Dir); os.IsNotExist(err) {
		t.Error("Clear of ns1 removed ns2 directory")
	}
}

func TestClear_EmptyNamespace(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool")

	_ = c.Set("k1", sample{Name: "a"}, time.Hour)

	if err := c.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	_, ok := c.Get("k1")
	if ok {
		t.Error("entry remains after Clear")
	}
}

func TestClear_NoDir(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("never-written"))

	err := c.Clear()
	if err != nil {
		t.Errorf("Clear on non-existent dir should not error, got: %v", err)
	}
}

func TestGetOrSet_Miss(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("gos"))

	called := false
	val, err := c.GetOrSet("k1", time.Hour, func() (sample, error) {
		called = true
		return sample{Name: "fetched"}, nil
	})
	if err != nil {
		t.Fatalf("GetOrSet: %v", err)
	}
	if !called {
		t.Error("fn was not called on miss")
	}
	if val.Name != "fetched" {
		t.Errorf("Name = %q, want %q", val.Name, "fetched")
	}

	val2, ok := c.Get("k1")
	if !ok {
		t.Fatal("value was not cached after GetOrSet")
	}
	if val2.Name != "fetched" {
		t.Errorf("cached Name = %q, want %q", val2.Name, "fetched")
	}
}

func TestGetOrSet_Hit(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("gos"))

	_ = c.Set("k1", sample{Name: "cached"}, time.Hour)

	called := false
	val, err := c.GetOrSet("k1", time.Hour, func() (sample, error) {
		called = true
		return sample{Name: "fresh"}, nil
	})
	if err != nil {
		t.Fatalf("GetOrSet: %v", err)
	}
	if called {
		t.Error("fn was called on cache hit")
	}
	if val.Name != "cached" {
		t.Errorf("Name = %q, want %q", val.Name, "cached")
	}
}

func TestGetOrSet_FnError(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("gos"))

	_, err := c.GetOrSet("k1", time.Hour, func() (sample, error) {
		return sample{}, fmt.Errorf("fetch failed")
	})
	if err == nil {
		t.Error("expected error from fn")
	}
}

func TestGetOrSet_FnSuccessSetFails(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("readonly"))

	os.MkdirAll(c.dir, 0o755)
	os.Chmod(c.dir, 0o444)
	defer os.Chmod(c.dir, 0o755)

	val, err := c.GetOrSet("k1", time.Hour, func() (sample, error) {
		return sample{Name: "fetched"}, nil
	})

	if val.Name != "fetched" {
		t.Errorf("Name = %q, want %q", val.Name, "fetched")
	}
	if err == nil {
		t.Error("expected error from Set failure")
	}
}

func TestGet_UnreadableFileNotDeleted(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("perm"))

	_ = c.Set("k1", sample{Name: "secret"}, time.Hour)

	path := c.filepath("k1")
	os.Chmod(path, 0o000)
	defer os.Chmod(path, 0o644)

	_, ok := c.Get("k1")
	if ok {
		t.Error("Get returned true for unreadable file")
	}

	// File should NOT be deleted (permission error, not corruption)
	os.Chmod(path, 0o644)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("unreadable file was deleted — should be preserved")
	}
}

func TestGetOrSet_EmptyKey(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("gos"))

	called := false
	_, err := c.GetOrSet("", time.Hour, func() (sample, error) {
		called = true
		return sample{Name: "should-not-be-called"}, nil
	})
	if called {
		t.Error("fn was called for empty key")
	}
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestConcurrentSetGet(t *testing.T) {
	testDir(t)
	c := New[sample]("test-tool", WithNamespace("concurrent"))

	const goroutines = 20
	done := make(chan struct{}, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer func() { done <- struct{}{} }()
			key := fmt.Sprintf("key-%d", i)
			val := sample{Name: fmt.Sprintf("val-%d", i)}
			if err := c.Set(key, val, time.Hour); err != nil {
				t.Errorf("Set(%q): %v", key, err)
				return
			}
			got, ok := c.Get(key)
			if !ok {
				t.Errorf("Get(%q) returned false", key)
				return
			}
			if got.Name != val.Name {
				t.Errorf("Get(%q).Name = %q, want %q", key, got.Name, val.Name)
			}
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}
}
