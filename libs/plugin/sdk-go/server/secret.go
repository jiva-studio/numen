package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
)

// SecretAPIKey is the secret a plugin declares in its manifest for the key of
// the service it calls.
const SecretAPIKey = "apiKey"

// Secret is one allowed secret with its value. Printing or logging a Secret with any verb shows
// its id, name and group and never its value, also when an encoder writes it; a plugin reads the
// value from the Value field.
type Secret struct{ ID, Name, Group, Value string }

// String describes the secret without its value.
func (s Secret) String() string {
	return fmt.Sprintf("{%s %s %s}", s.ID, s.Name, s.Group)
}

// GoString describes the secret without its value.
func (s Secret) GoString() string {
	return fmt.Sprintf("server.Secret{ID:%q, Name:%q, Group:%q}", s.ID, s.Name, s.Group)
}

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
func (s Secret) LogValue() slog.Value {
	return slog.GroupValue(slog.String("id", s.ID), slog.String("name", s.Name), slog.String("group", s.Group))
}

// Secrets holds, for each active declaration of the plugin, the allowed secrets in the order the
// host chose.
type Secrets map[string][]Secret

// ReadSecrets returns the secrets a Configure call passes for a declaration, and whether the call
// carries that declaration. An empty list means no secret, and the plugin discards the ones it has.
// The list holds the secrets the host could read: one it could not read is absent, so the plugin
// does not keep it.
func ReadSecrets(secrets Secrets, name string) ([]Secret, bool) {
	list, found := secrets[name]
	if !found {
		return nil, false
	}
	return list, true
}

// MarshalText encodes the secret as the text String returns, without its value.
func (s Secret) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// MarshalJSON encodes the secret as the JSON text String returns, without its value.
func (s Secret) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}
