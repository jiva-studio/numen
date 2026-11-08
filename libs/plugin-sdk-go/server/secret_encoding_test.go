package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// An encoder shows a secret without its value, alone and inside a slice, a map or a struct.
func TestASecretEncodedAsJSONShowsNoValue(t *testing.T) {
	one := Secret{ID: "s0000000a", Name: "Work", Group: "openrouter", Value: redactedValue}
	encoded := map[string]any{
		"alone":   one,
		"slice":   []Secret{one},
		"map":     map[string]Secret{"a": one},
		"struct":  struct{ Key Secret }{one},
		"secrets": Secrets{SecretAPIKey: {one}},
	}
	for name, input := range encoded {
		raw, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), redactedValue) {
			t.Errorf("%s encodes the value: %s", name, raw)
		}
	}
	if raw, err := one.MarshalText(); err != nil || strings.Contains(string(raw), redactedValue) {
		t.Errorf("MarshalText = %q, %v, want no value", raw, err)
	}
}

func TestASecretInAListLoggedAsJSONShowsNoValue(t *testing.T) {
	one := Secret{ID: "s0000000a", Name: "Work", Group: "openrouter", Value: redactedValue}
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, nil))
	logger.Info("secrets", "list", []Secret{one}, "all", Secrets{SecretAPIKey: {one}})
	if strings.Contains(out.String(), redactedValue) {
		t.Errorf("the log holds the value: %s", out.String())
	}
}
