package service

import (
	"encoding/json"
	"io"
	"testing"
)

// encodeJSON writes value as JSON to w and reports a failure to t.
func encodeJSON(t testing.TB, w io.Writer, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encode the response: %v", err)
	}
}
