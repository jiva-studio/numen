package connection

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
)

// List reads the connections from the list, without any value of a secret.
type List struct{ operation }

// Execute returns the connections in list order. A secret counts as set only when its entry in the
// key store holds the fingerprint of the type and the address the connection has now.
func (u *List) Execute(ctx context.Context) ([]Entry, error) {
	list, err := u.repository.ListConnections()
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(list))
	for _, one := range list {
		entry, err := u.buildEntry(ctx, one)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// ListTypes reads the kinds of connection the installed plugins provide.
type ListTypes struct{ operation }

// Execute returns every declared type, the unavailable ones included.
func (u *ListTypes) Execute() []connection.Type {
	return u.types.ListTypes()
}
