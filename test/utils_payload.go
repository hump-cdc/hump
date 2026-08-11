package test

import (
	"encoding/json"
	"fmt"
)

// payload is the decoded default CDC payload produced by
// internal/port/sink/templ/payload.tmpl. Before/After/Source are zero when
// absent: Before is sent for update/delete, After for insert/update.
type payload struct {
	Before map[string]any `json:"before"`
	After  map[string]any `json:"after"`
	Source struct {
		Version   string `json:"version"`
		Connector string `json:"connector"`
		Name      string `json:"name"`
		DB        string `json:"db"`
		Schema    string `json:"schema"`
		Table     string `json:"table"`
		TsMs      int64  `json:"ts_ms"`
	} `json:"source"`
	Op   string `json:"op"` // c (insert) / u (update) / d (delete) / t (truncate)
	TsMs int64  `json:"ts_ms"`
}

// payloadDecode unmarshals a default CDC payload into a payload.
func payloadDecode(data []byte) (payload, error) {
	var p payload
	if err := json.Unmarshal(data, &p); err != nil {
		return payload{}, err
	}
	return p, nil
}

// stringify renders a JSON-decoded value as a stable string for
// comparison. Scalars use %v (so float64(1) → "1", matching int PKs);
// maps/slices use their compact JSON form (so a jsonb column round-trips to its
// canonical text).
func stringify(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case map[string]any, []any:
		b, _ := json.Marshal(t)
		return string(b)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// val extracts a numeric value (JSON numbers decode to float64).
func val(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}
