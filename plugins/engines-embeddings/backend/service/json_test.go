package service

import (
	"encoding/json"
	"io"
	"testing"
)

func encodeJSON(tb testing.TB, w io.Writer, value any) {
	tb.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		tb.Errorf("encode the response: %v", err)
	}
}
