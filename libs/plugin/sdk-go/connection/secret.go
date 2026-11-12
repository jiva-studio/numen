package connection

import (
	"fmt"
	"io"
	"log/slog"
)

const redacted = "[secret]"

// Secret is the value of a secret field of a connection. Printing, logging or encoding a Secret shows
// no value; a provider reads the value from the Value field.
type Secret struct{ Value string }

// String describes the secret without its value.
func (Secret) String() string { return redacted }

// GoString describes the secret without its value.
func (Secret) GoString() string { return "connection.Secret{" + redacted + "}" }

// Format prints the secret without its value for every verb.
func (s Secret) Format(f fmt.State, verb rune) {
	text := s.String()
	if verb == 'v' && f.Flag('#') {
		text = s.GoString()
	}
	if _, err := io.WriteString(f, text); err != nil {
		return
	}
}

// LogValue describes the secret to a structured logger without its value.
func (s Secret) LogValue() slog.Value { return slog.StringValue(s.String()) }

// MarshalText encodes the secret as the text String returns, without its value.
func (s Secret) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

// MarshalJSON encodes the secret as the JSON text String returns, without its value.
func (s Secret) MarshalJSON() ([]byte, error) { return []byte(`"` + redacted + `"`), nil }
