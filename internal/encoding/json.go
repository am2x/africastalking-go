package encoding

import (
	"encoding/json"
	"fmt"
)

// EncodeJSON marshals v for a JSON endpoint. It takes any rather than a
// concrete type so that this package never needs to import the root package,
// which would be an import cycle.
func EncodeJSON(v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte{}, fmt.Errorf("encoding: marshal json: %w", err)
	}

	return data, nil
}
