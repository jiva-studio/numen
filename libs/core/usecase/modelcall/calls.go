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
// goes to the next row on the first failure of a connection. An error of a provider that is neither failure
// kind counts as a service that is unreachable. A row is skipped, and its reason is in the error
// when every row fails, if its connection is gone, its type is not provided or lacks the operation, its key is
// missing, it is set aside, or its connection stayed busy for the wait its priority allows.
type Calls struct {
	connections *connection.Connections
	rows        connectionport.Rows
	source      connectionport.ProviderSource
	clock       machineport.Clock
	queues      *queues
	notices     connectionport.Notices
	cooldown    *cooldown
	noticeLog   *noticeLog
}

// NewCalls returns the calls over the connections, the rows of the plugins and the providers the source
// opens. wait times the calls that wait for a slot of their connection. notices is called when a connection was set aside and a row after it answered.
func NewCalls(
	connections *connection.Connections, rows connectionport.Rows, source connectionport.ProviderSource,
	clock machineport.Clock, wait machineport.Wait, notices connectionport.Notices,
) *Calls {
	return &Calls{
		connections: connections, rows: rows, source: source, clock: clock, queues: newQueues(wait), notices: notices,
		cooldown: newCooldown(), noticeLog: newNoticeLog(),
	}
}
