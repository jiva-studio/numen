package modelcall

import (
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

const (
	// DefaultCooldown is how long a connection is set aside when the service gave no wait.
	DefaultCooldown = 60 * time.Second
	// MaxCooldown is the longest a connection is set aside for a wait the service asked for.
	MaxCooldown = 10 * time.Minute
)

// Calls carries out the operations plugins ask for. It reads the rows of the calling plugin in order and
// goes to the next row on the first failure of a connection. An error of a provider that is neither failure
// kind counts as a service that is unreachable. A row is skipped, and its reason is in the error
// when every row fails, if its connection is gone, its type is not provided or lacks the operation, its key is
// missing, it is set aside, its connection stayed busy for the wait its priority allows, or its connection
// changed during the call twice in a row. A connection that changed once during the call is read again. When a
// later row answers, notices receives each connection set aside before it, except a row that discovery made
// rather than the person.
type Calls struct {
	resolve     connectionport.ResolveConnection
	readSecrets connectionport.ReadConnectionSecrets
	rows        connectionport.ListRows
	listOptions connectionport.ListRowOptions
	source      connectionport.ProviderSource
	clock       machineport.Clock
	queues      *queues
	notices     connectionport.ReportSetAside
	cooldown    *cooldown
}

// Connections is where the calls find a connection, its secrets, the rows of the plugins that name it and
// the values of the row fields of its type.
type Connections struct {
	Resolve     connectionport.ResolveConnection
	ReadSecrets connectionport.ReadConnectionSecrets
	Rows        connectionport.ListRows
	ListOptions connectionport.ListRowOptions
}

// NewCalls returns the calls over the connections and the providers the source
// opens. wait times the calls that wait for a slot of their connection. notices is called when a connection was set aside and a row after it answered.
func NewCalls(
	connections Connections, source connectionport.ProviderSource, clock machineport.Clock, wait Wait, notices connectionport.ReportSetAside,
) *Calls {
	return &Calls{
		resolve: connections.Resolve, readSecrets: connections.ReadSecrets, rows: connections.Rows, listOptions: connections.ListOptions, source: source, clock: clock, queues: newQueues(wait), notices: notices,
		cooldown: newCooldown(),
	}
}
