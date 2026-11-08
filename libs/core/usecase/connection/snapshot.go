package connection

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
)

// ErrSecretMissing is a required secret field with no value stored for the address the connection has.
var ErrSecretMissing = errors.New("a required secret has no value")

// Resolved is a connection as one call to its provider reads it: the fields as text, with the fixed
// values of the type, and no value of a secret.
type Resolved struct {
	ID          string
	Name        string
	Revision    int
	PluginID    string
	Declaration connectionport.ConnectionType
	Fields      map[string]string
	fingerprint string
}

// NewCall makes the call to the provider of the connection with the secrets read for it, the model and
// the reasoning level of the row.
func (r Resolved) NewCall(secrets map[string]string, model, reasoning string) connectionport.ProviderCall {
	return connectionport.ProviderCall{
		Type:      r.Declaration.ID,
		Fields:    maps.Clone(r.Fields),
		Secrets:   maps.Clone(secrets),
		Model:     model,
		Reasoning: reasoning,
	}
}

// Resolve reads a connection and its type as a call to the provider needs them.
type Resolve struct{ operation }

// Execute returns the connection with the id. It fails with ErrNotFound for an id not in the list and
// with ErrTypeUnavailable where no enabled plugin provides the type.
func (u *Resolve) Execute(connectionID string) (Resolved, error) {
	list, err := u.repository.ListConnections()
	if err != nil {
		return Resolved{}, err
	}
	position := getIndex(list, connectionID)
	if position < 0 {
		return Resolved{}, newNotFoundError(connectionID)
	}
	one := list[position]
	found, err := u.getAvailableType(one.TypeID)
	if err != nil {
		return Resolved{}, err
	}
	return Resolved{
		ID: one.ID, Name: one.Name, Revision: one.Revision, PluginID: found.PluginID, Declaration: found.Declaration,
		Fields:      formatFields(found.Declaration, one.Fields),
		fingerprint: getFingerprint(one.TypeID, getAddress(found.Declaration, one.Fields)),
	}, nil
}

func formatFields(declared connectionport.ConnectionType, stored map[string]any) map[string]string {
	fields := map[string]string{}
	for _, field := range declared.Fields {
		if field.Type == secretFieldType {
			continue
		}
		if field.FixedValue != nil {
			fields[field.Key] = *field.FixedValue
			continue
		}
		if value, isSet := stored[field.Key]; isSet {
			fields[field.Key] = formatValue(value)
		}
	}
	return fields
}

func formatValue(value any) string {
	if number, isNumber := toFloat(value); isNumber {
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
	if text, isText := value.(string); isText {
		return text
	}
	return fmt.Sprint(value)
}

// ReadSecrets reads the secret fields of a resolved connection from the key store.
type ReadSecrets struct{ operation }

// Execute returns the value of each secret field that has one for the address the connection has. A
// required secret with no value is ErrSecretMissing. It fails with ErrStaleRevision when the connection
// is gone or its revision is no longer the one resolved, so a key is never returned for a connection
// that changed while it was read.
func (u *ReadSecrets) Execute(ctx context.Context, resolved Resolved) (map[string]string, error) {
	secrets := map[string]string{}
	for _, field := range resolved.Declaration.Fields {
		if field.Type != secretFieldType {
			continue
		}
		entry, isFound, err := u.secrets().read(ctx, resolved.ID, field.Key)
		if err != nil {
			return nil, err
		}
		if value, isStored := readSecret(entry, resolved.fingerprint); isFound && isStored {
			secrets[field.Key] = value
			continue
		}
		if field.IsRequired {
			return nil, fmt.Errorf("%w: %s", ErrSecretMissing, field.Key)
		}
	}
	list, err := u.repository.ListConnections()
	if err != nil {
		return nil, err
	}
	position := getIndex(list, resolved.ID)
	if position < 0 || list[position].Revision != resolved.Revision {
		return nil, newStaleRevisionError(resolved.ID)
	}
	return secrets, nil
}

func readSecret(entry, fingerprint string) (string, bool) {
	if !hasFingerprint(entry, fingerprint) {
		return "", false
	}
	_, value, _ := strings.Cut(entry, fingerprintSeparator)
	return value, true
}
