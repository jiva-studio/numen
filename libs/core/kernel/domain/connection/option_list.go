package connection

import "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

// OptionList is the options a provider listed for one row field of a connection, with the values of the
// row fields declared before it and the operations it was asked with, and what of the connection the
// listing depends on.
type OptionList struct {
	ConnectionID  string
	Field         string
	RowOptions    map[string]string
	Operations    []string
	Revision      int
	PluginVersion string
	Fields        map[string]string
	Options       []modelcall.Option
}
