package modelcall

import (
	"time"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/usecase/connection"
)

const (
	// DefaultCooldown is how long a connection is set aside when the service gave no wait.
	DefaultCooldown = 60 * time.Second
	// MaxCooldown is the longest a connection is set aside for a wait the service asked for.
	MaxCooldown = 10 * time.Minute
)

// Calls carries out the operations plugins ask for. It reads the rows of the calling plugin in order and
// goes to the next row on the first failure of a connection.
type Calls struct {
	connections *connection.Connections
	rows        connectionport.Rows
	source      connectionport.ProviderSource
	clock       machineport.Clock
	notices     connectionport.Notices
	cooldown    *cooldown
	told        *told
}

// NewCalls returns the calls over the connections, the rows of the plugins and the providers the source
// opens. notices is told of a connection set aside when a row after it answers.
func NewCalls(
	connections *connection.Connections, rows connectionport.Rows, source connectionport.ProviderSource,
	clock machineport.Clock, notices connectionport.Notices,
) *Calls {
	return &Calls{
		connections: connections, rows: rows, source: source, clock: clock, notices: notices,
		cooldown: newCooldown(), told: newTold(),
	}
}
