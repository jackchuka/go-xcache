package xcache

import "encoding/json"

// Codec defines how cache entries are serialized and deserialized.
type Codec interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
	Extension() string // must include leading dot: ".json", ".gob", etc.
}

// JSONCodec serializes entries as indented JSON.
type JSONCodec struct{}

func (JSONCodec) Marshal(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func (JSONCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (JSONCodec) Extension() string {
	return ".json"
}
