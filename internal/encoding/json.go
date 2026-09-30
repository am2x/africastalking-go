package encoding

import (
	"encoding/json"
	"fmt"
)

func EncodeJSON(v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte{}, fmt.Errorf("encoding: marshal json: %w", err)
	}

	return data, nil
}
