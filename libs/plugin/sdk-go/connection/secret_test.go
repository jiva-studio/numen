package connection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

const redactedValue = "sk-test-0123456789"

func TestASecretShowsNoValueForAnyVerb(t *testing.T) {
	one := Secret{Value: redactedValue}
	for _, printed := range []string{
		fmt.Sprint(one), fmt.Sprintf("%v", one), fmt.Sprintf("%+v", one), fmt.Sprintf("%#v", one),
		fmt.Sprintf("%s", one), fmt.Sprintf("%q", one), fmt.Sprintf("%v", &one), fmt.Sprintf("%v", map[string]Secret{"apiKey": one}),
	} {
		if strings.Contains(printed, redactedValue) {
			t.Errorf("a secret printed its value: %s", printed)
		}
	}
}

func TestASecretEncodedAsJSONShowsNoValue(t *testing.T) {
	one := Secret{Value: redactedValue}
	for name, input := range map[string]any{
		"alone": one, "slice": []Secret{one}, "map": map[string]Secret{"apiKey": one},
		"call": Call{Secrets: map[string]Secret{"apiKey": one}},
	} {
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

func TestASecretLoggedShowsNoValue(t *testing.T) {
	one := Secret{Value: redactedValue}
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, nil))
	logger.Info("call", "secret", one, "secrets", map[string]Secret{"apiKey": one}, "call", Call{Secrets: map[string]Secret{"apiKey": one}})
	if strings.Contains(out.String(), redactedValue) {
		t.Errorf("the log holds the value: %s", out.String())
	}
}

func TestASecretKeepsItsValueForTheProvider(t *testing.T) {
	if got := (Secret{Value: redactedValue}).Value; got != redactedValue {
		t.Errorf("Value = %q, want %q", got, redactedValue)
	}
}
