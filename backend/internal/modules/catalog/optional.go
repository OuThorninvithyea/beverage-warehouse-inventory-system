package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type OptionalString struct {
	Set   bool
	Value *string
}

func (o *OptionalString) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(data, []byte("null")) {
		o.Value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("optional string must be a string or null: %w", err)
	}
	o.Value = &value
	return nil
}
