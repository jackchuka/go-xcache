package xcache

import (
	"testing"
	"time"
)

type testStruct struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestJSONCodec_RoundTrip(t *testing.T) {
	codec := JSONCodec{}

	original := entry[testStruct]{
		Key:       "test-key",
		ExpiresAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC),
		Value:     testStruct{Name: "hello", Count: 42},
	}

	data, err := codec.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got entry[testStruct]
	if err := codec.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Key != original.Key {
		t.Errorf("Key = %q, want %q", got.Key, original.Key)
	}
	if !got.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, original.ExpiresAt)
	}
	if got.Value.Name != original.Value.Name || got.Value.Count != original.Value.Count {
		t.Errorf("Value = %+v, want %+v", got.Value, original.Value)
	}
}

func TestJSONCodec_Extension(t *testing.T) {
	codec := JSONCodec{}
	if ext := codec.Extension(); ext != ".json" {
		t.Errorf("Extension() = %q, want %q", ext, ".json")
	}
}

func TestJSONCodec_NestedStructs(t *testing.T) {
	type inner struct {
		Tags []string          `json:"tags"`
		Meta map[string]string `json:"meta"`
	}

	codec := JSONCodec{}

	original := entry[inner]{
		Key:       "nested",
		ExpiresAt: time.Now().Add(time.Hour),
		Value: inner{
			Tags: []string{"a", "b"},
			Meta: map[string]string{"k": "v"},
		},
	}

	data, err := codec.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got entry[inner]
	if err := codec.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(got.Value.Tags) != 2 || got.Value.Tags[0] != "a" {
		t.Errorf("Tags = %v, want [a b]", got.Value.Tags)
	}
	if got.Value.Meta["k"] != "v" {
		t.Errorf("Meta = %v, want map[k:v]", got.Value.Meta)
	}
}
