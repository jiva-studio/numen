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

// ReportSetAside writes a line for the connection, the reason and the connection that answered. A failed write
// is dropped: the line is a log, and there is nowhere else to report the failure.
func (n LogNotices) ReportSetAside(setAside connection.SetAside) {
	line := fmt.Sprintf("connection %s (%s) was set aside: %s; %s answered\n", setAside.ConnectionName, setAside.ConnectionID, setAside.Reason, setAside.NextName)
	if _, err := n.out.WriteString(line); err != nil {
		return
	}
}
