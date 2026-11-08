package service

import (
	"encoding/json"
	"io"
	"testing"
)

func encodeJSON(t testing.TB, w io.Writer, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encode the response: %v", err)
	}
}
