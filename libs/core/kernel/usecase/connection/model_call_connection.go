package connection

import "github.com/jiva-studio/numen/libs/core/kernel/domain/connection"

// RowLister returns the rows a plugin calls an operation through, in the order of use.
type RowLister func(pluginID, operation string) ([]connection.Row, error)

// ReportSetAside tells the person that a connection was set aside for a reason and a row after it answered.
type ReportSetAside func(setAside connection.SetAside)
