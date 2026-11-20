package connection

import "github.com/jiva-studio/numen/libs/core/kernel/domain/connection"

// RowSettings reads the settings of rows of the plugins.
type RowSettings interface {
	// ReadRowSetting returns the setting of rows of the plugin for an operation, with the rows that name an
	// existing connection. A plugin with no such setting shown has no row and does not discover, whatever
	// it is granted.
	ReadRowSetting(pluginID, operation string) (connection.RowSetting, error)
}
