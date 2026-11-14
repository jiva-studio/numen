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

// Parts are what the use cases are built over: the list, the registry of types, the key store, the rows that
// name a connection, the connections detected, listed after the stored ones, and the titles of the operations.
type Parts struct {
	Repository connectionport.Repository
	Types      connectionport.Types
	Keys       pluginport.KeyStore
	Uses       connectionport.Uses
	Detected   connectionport.Detected
	Titles     connectionport.OperationTitles
}

// New returns the use cases over the parts. newID makes the id of a new connection.
func New(parts Parts, newID func() (string, error)) *Connections {
	deps := connection.NewDeps(parts.Repository, parts.Types, parts.Keys, parts.Uses, parts.Detected)
	return &Connections{
		Manage: Manage{
			Create: connection.NewCreate(deps, newID), Update: connection.NewUpdate(deps), Delete: connection.NewDelete(deps),
			List: connection.NewList(deps), ListTypes: connection.NewListTypes(deps, parts.Titles),
		},
		Read: Read{Resolve: connection.NewResolve(deps), ReadSecrets: connection.NewReadSecrets(deps)},
	}
}
