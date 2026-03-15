package xcache

import "testing"

func TestSHA256KeyFunc_Consistent(t *testing.T) {
	result1 := SHA256KeyFunc("owner/repo")
	result2 := SHA256KeyFunc("owner/repo")
	if result1 != result2 {
		t.Errorf("not consistent: %q != %q", result1, result2)
	}
}

func TestSHA256KeyFunc_FilenameSafe(t *testing.T) {
	result := SHA256KeyFunc("owner/repo:with*special?chars")
	for _, c := range result {
		if c == '/' || c == '\\' || c == ':' || c == '*' || c == '?' {
			t.Errorf("unsafe character %q in result %q", string(c), result)
		}
	}
}

func TestSHA256KeyFunc_DifferentKeys(t *testing.T) {
	r1 := SHA256KeyFunc("key1")
	r2 := SHA256KeyFunc("key2")
	if r1 == r2 {
		t.Error("different keys produced same hash")
	}
}

func TestSafeNameKeyFunc_SimpleKey(t *testing.T) {
	result := SafeNameKeyFunc("my-simple-key")
	if result != "my-simple-key" {
		t.Errorf("SafeNameKeyFunc(%q) = %q, want %q", "my-simple-key", result, "my-simple-key")
	}
}

func TestSafeNameKeyFunc_ReplacesUnsafeChars(t *testing.T) {
	result := SafeNameKeyFunc("owner/repo")
	if result != "owner_repo" {
		t.Errorf("SafeNameKeyFunc(%q) = %q, want %q", "owner/repo", result, "owner_repo")
	}
}

func TestSafeNameKeyFunc_AllUnsafeChars(t *testing.T) {
	result := SafeNameKeyFunc(`a/b\c:d*e?f"g<h>i|j`)
	expected := "a_b_c_d_e_f_g_h_i_j"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestSafeNameKeyFunc_LongKeyFallsBackToHash(t *testing.T) {
	longKey := ""
	for i := 0; i < 201; i++ {
		longKey += "a"
	}
	result := SafeNameKeyFunc(longKey)
	if len(result) != 64 {
		t.Errorf("long key should fall back to SHA-256, got length %d", len(result))
	}
}
