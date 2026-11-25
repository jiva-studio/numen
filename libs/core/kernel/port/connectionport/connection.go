package connectionport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
)

// Repository is the ordered list of connections kept in the settings file.
type Repository interface {
	// ListConnections returns the valid connections in list order.
	ListConnections() ([]connection.Connection, error)
	// ChangeConnections reads the list, applies change and writes the result as one step. Nothing is
	// written when change fails or leaves the list as it was.
	ChangeConnections(change func([]connection.Connection) ([]connection.Connection, error)) error
}

// Types is the registry of the kinds of connection the installed plugins provide.
type Types interface {
	// ListTypes returns the types of the enabled plugins, in the order of the plugin ids and then of the manifest.
	ListTypes() []connection.Type
	// GetType returns the type with the id, and false where no installed plugin declares it.
	GetType(typeID string) (connection.Type, bool)
}

// ResolveConnection reads a connection and its type as a call to the provider needs them. It fails with
// connection.ErrNotFound and connection.ErrTypeUnavailable.
type ResolveConnection func(connectionID string) (connection.Resolution, error)

// DetectedConnections is the connections the providers report ready on this machine through connection.OperationDetected.
type DetectedConnections interface {
	// ListDetected returns the connections the providers last reported for the available types, in the order
	// of the types. It does not wait for a provider.
	ListDetected() []connection.Connection
	// IsPending reports whether the type is available, serves detection and its provider has not answered yet
	// in the version of its plugin installed now; an answer kept from an earlier start counts.
	IsPending(typeID string) bool
	// Detect asks the provider of each available type of the plugin that serves detection and waits for the
	// answers.
	Detect(ctx context.Context, pluginID string)
	// Refresh asks every provider not asked lately again in the background.
	Refresh()
}
