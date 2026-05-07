package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type FlexibleInt64 int64

func (v *FlexibleInt64) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*v = 0
		return nil
	}

	if len(trimmed) >= 2 && trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
		var value string
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return fmt.Errorf("unmarshal string int64: %w", err)
		}
		value = strings.TrimSpace(value)
		if value == "" {
			*v = 0
			return nil
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("parse string int64 %q: %w", value, err)
		}
		*v = FlexibleInt64(parsed)
		return nil
	}

	parsed, err := strconv.ParseInt(string(trimmed), 10, 64)
	if err != nil {
		return fmt.Errorf("parse numeric int64 %q: %w", string(trimmed), err)
	}
	*v = FlexibleInt64(parsed)
	return nil
}

func (v FlexibleInt64) Int64() int64 {
	return int64(v)
}
