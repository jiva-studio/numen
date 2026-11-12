// Package connectiontest builds the use cases on connections as a test uses them, all over the same parts.
package connectiontest

import (
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
)

// Manage is the use cases a window works the list of connections with.
type Manage struct {
	Create    *connection.Create
	Update    *connection.Update
	Delete    *connection.Delete
	List      *connection.List
	ListTypes *connection.ListTypes
}

// Read is the use cases that give a call what it needs of a connection.
type Read struct {
	Resolve     *connection.Resolve
	ReadSecrets *connection.ReadSecrets
}

// Connections is every use case on connections.
type Connections struct {
	Manage
	Read
}

// New returns the use cases over the list, the registry of types and the key store. newID makes the id of
// a new connection.
func New(
	repository connectionport.Repository, types connectionport.Types, keys pluginport.KeyStore, uses connectionport.Uses,
	newID func() (string, error),
) *Connections {
	deps := connection.NewDeps(repository, types, keys, uses)
	return &Connections{
		Manage: Manage{
			Create: connection.NewCreate(deps, newID), Update: connection.NewUpdate(deps), Delete: connection.NewDelete(deps),
			List: connection.NewList(deps), ListTypes: connection.NewListTypes(deps),
		},
		Read: Read{Resolve: connection.NewResolve(deps), ReadSecrets: connection.NewReadSecrets(deps)},
	}
}
