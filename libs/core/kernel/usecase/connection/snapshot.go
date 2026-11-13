package connection

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Resolve reads a connection and its type as a call to the provider needs them.
type Resolve struct{ operation }

// Execute returns the connection with the id. A detected connection of a type whose provider has not reported
// what it detects yet is asked for first. It fails with connection.ErrNotFound for an id not in the list and
// with connection.ErrTypeUnavailable where no enabled plugin provides the type.
func (u *Resolve) Execute(connectionID string) (connection.Resolved, error) {
	typeID, _, _ := connection.SplitDetectedID(connectionID)
	list, err := u.listConnectionsAfterDetection([]string{typeID})
	if err != nil {
		return connection.Resolved{}, err
	}
	position := getIndex(list, connectionID)
	if position < 0 {
		return connection.Resolved{}, newNotFoundError(connectionID)
	}
	one := list[position]
	found, err := u.getAvailableType(one.TypeID)
	if err != nil {
		return connection.Resolved{}, err
	}
	return connection.Resolved{
		ID: one.ID, Name: one.Name, Revision: one.Revision, PluginID: found.PluginID, Declaration: found.Declaration,
		Fields:           formatFields(found.Declaration, one.Fields),
		ParallelRequests: getParallelRequests(found.Declaration, one.Fields),
		Fingerprint:      getFingerprint(one.TypeID, getAddress(found.Declaration, one.Fields)),
	}, nil
}

func formatFields(declared manifest.ConnectionType, stored map[string]any) map[string]string {
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
// required secret with no value is connection.ErrSecretMissing. It fails with connection.ErrStaleRevision when the connection
// is gone or its revision is no longer the one resolved, so a key is never returned for a connection
// that changed while it was read.
func (u *ReadSecrets) Execute(ctx context.Context, resolved connection.Resolved) (map[string]string, error) {
	secrets := map[string]string{}
	for _, field := range resolved.Declaration.Fields {
		if field.Type != secretFieldType {
			continue
		}
		entry, isFound, err := u.secrets().read(ctx, resolved.ID, field.Key)
		if err != nil {
			return nil, err
		}
		if value, isStored := readSecret(entry, resolved.Fingerprint); isFound && isStored {
			secrets[field.Key] = value
			continue
		}
		if field.IsRequired {
			return nil, fmt.Errorf("%w: %s", connection.ErrSecretMissing, field.Key)
		}
	}
	list, err := u.listConnections()
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

func getParallelRequests(declared manifest.ConnectionType, stored map[string]any) int {
	if limit, isSet := toLimit(stored[manifest.ConnectionParallelRequests]); isSet {
		return limit
	}
	if field, isDeclared := declared.GetField(manifest.ConnectionParallelRequests); isDeclared {
		limit, _ := toLimit(field.Default)
		return limit
	}
	return 0
}

func toLimit(value any) (int, bool) {
	number, isNumber := toFloat(value)
	if !isNumber || number < 1 || number != math.Trunc(number) {
		return 0, false
	}
	return int(number), true
}
