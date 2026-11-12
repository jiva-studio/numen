// Package connectionlog writes what the core did with the connections of the person to a log.
package connectionlog

import (
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
)

// LogNotices writes the connections set aside to a log, for a window that shows no list of tasks.
type LogNotices struct{ out io.Writer }

// NewLogNotices returns notices that write to out.
func NewLogNotices(out io.Writer) LogNotices { return LogNotices{out: out} }

// ReportSetAside writes a line for the connection and the reason.
func (n LogNotices) ReportSetAside(connectionID, connectionName string, reason connection.FailureReason) {
	if _, err := fmt.Fprintf(n.out, "connection %s (%s) was set aside: %s\n", connectionName, connectionID, reason); err != nil {
		return
	}
}
