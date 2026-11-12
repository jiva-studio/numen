// Package connectionlog writes what the core did with the connections of the person to the error output of the process.
package connectionlog

import (
	"fmt"
	"os"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
)

// LogNotices writes the connections set aside to a file, for a window that shows no list of tasks.
type LogNotices struct{ out *os.File }

// NewLogNotices returns notices that write to the error output of the process.
func NewLogNotices() LogNotices { return LogNotices{out: os.Stderr} }

// ReportSetAside writes a line for the connection and the reason. A failed write is dropped: the line
// is a log, and there is nowhere else to report the failure.
func (n LogNotices) ReportSetAside(connectionID, connectionName string, reason connection.FailureReason) {
	if _, err := n.out.WriteString(fmt.Sprintf("connection %s (%s) was set aside: %s\n", connectionName, connectionID, reason)); err != nil {
		return
	}
}
