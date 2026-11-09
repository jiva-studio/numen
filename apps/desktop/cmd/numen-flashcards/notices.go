package main

import (
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
)

// logNotices writes the connections set aside to the log, because this window shows no list of tasks.
type logNotices struct{ out io.Writer }

func (n logNotices) ReportSetAside(connectionID, connectionName string, reason connectionport.FailureReason) {
	if _, err := fmt.Fprintf(n.out, "%s: connection %s (%s) was set aside: %s\n", appName, connectionName, connectionID, reason); err != nil {
		return
	}
}
