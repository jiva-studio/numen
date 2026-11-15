package connection

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

const redactedValue = "sk-test-0123456789"

type failingFormatState struct {
	calls int
}

func (f *failingFormatState) Write([]byte) (int, error) {
	f.calls++
	return 0, errors.New("write error")
}

func (*failingFormatState) Width() (int, bool)     { return 0, false }
func (*failingFormatState) Precision() (int, bool) { return 0, false }
func (*failingFormatState) Flag(int) bool          { return false }

func TestASecretShowsNoValueForAnyVerb(t *testing.T) {
	one := Secret{Value: redactedValue}
	for _, printed := range []string{
		fmt.Sprint(one), fmt.Sprintf("%v", one), fmt.Sprintf("%+v", one), fmt.Sprintf("%#v", one),
		fmt.Sprintf("%s", one), fmt.Sprintf("%#s", one), fmt.Sprintf("%q", one), fmt.Sprintf("%v", &one), fmt.Sprintf("%v", map[string]Secret{"apiKey": one}),
	} {
		if strings.Contains(printed, redactedValue) || !strings.Contains(printed, redacted) {
			t.Errorf("a secret printed %q", printed)
		}
	}
	if one.String() != redacted || one.GoString() != "connection.Secret{[secret]}" {
		t.Errorf("String = %q, GoString = %q", one.String(), one.GoString())
	}
	if got := fmt.Sprintf("%#v", one); got != "connection.Secret{[secret]}" {
		t.Errorf("%%#v = %q", got)
	}
	if got := fmt.Sprintf("%v", one); got != redacted {
		t.Errorf("%%v = %q", got)
	}
	if got := fmt.Sprintf("%#s", one); got != redacted {
		t.Errorf("%%#s = %q", got)
	}
	fs := &failingFormatState{}
	one.Format(fs, 'v')
	if fs.calls != 1 {
		t.Errorf("Format write calls = %d, want 1", fs.calls)
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
		if strings.Contains(string(raw), redactedValue) || !strings.Contains(string(raw), redacted) {
			t.Errorf("%s encodes: %s", name, raw)
		}
	}
	if raw, err := one.MarshalText(); err != nil || string(raw) != redacted {
		t.Errorf("MarshalText = %q, %v, want %q", raw, err, redacted)
	}
}

func TestASecretLoggedShowsNoValue(t *testing.T) {
	one := Secret{Value: redactedValue}
	if got := one.LogValue().String(); got != redacted {
		t.Errorf("LogValue = %q, want %q", got, redacted)
	}
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, nil))
	logger.Info("call", "secret", one, "secrets", map[string]Secret{"apiKey": one}, "call", Call{Secrets: map[string]Secret{"apiKey": one}})
	if strings.Contains(out.String(), redactedValue) || !strings.Contains(out.String(), redacted) {
		t.Errorf("the log holds: %s", out.String())
	}
}

func TestASecretKeepsItsValueForTheProvider(t *testing.T) {
	if got := (Secret{Value: redactedValue}).Value; got != redactedValue {
		t.Errorf("Value = %q, want %q", got, redactedValue)
	}
}
