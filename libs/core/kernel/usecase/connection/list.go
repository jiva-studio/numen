package connection

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// List reads the connections from the list, without any value of a secret.
type List struct{ operation }

// Execute returns the stored connections in list order, followed by the built-in ones. A secret counts as set only when its entry in the
// key store holds the fingerprint of the type and the address the connection has now.
func (u *List) Execute(ctx context.Context) ([]Entry, error) {
	list, err := u.listConnections()
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

// TypeList is the kinds of connection and the titles the plugins give the operations of the kinds. Types
// holds the types of the enabled plugins; Operations holds one title for each titled
// operation, in the order of the plugin ids.
type TypeList struct {
	Types      []connection.Type
	Operations []manifest.Operation
}

// ListTypes reads the kinds of connection the installed plugins provide and the titles of their operations.
type ListTypes struct {
	operation
	titles connectionport.OperationTitles
}

// Execute returns the kinds of connection and the titles of the operations; no titles where the use case
// has no source of titles.
func (u *ListTypes) Execute() TypeList {
	list := TypeList{Types: u.types.ListTypes()}
	if u.titles != nil {
		list.Operations = u.titles.ListOperationTitles()
	}
	return list
}
