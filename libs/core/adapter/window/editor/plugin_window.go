package editor

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"

	noteport "github.com/jiva-studio/numen/libs/core/usecase/note/notecontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	vaultport "github.com/jiva-studio/numen/libs/core/usecase/vault/vaultcontract"
)

var errPluginWindowNotImplemented = errors.New("the plugin window is not implemented")

// pluginWindow is the window of the state, as the plugins that call the host move it and read its tabs.
type pluginWindow struct{}

// NewPluginWindow is the window the state serves, as the plugins move it to a note through the
// queries and to a file of the vault through the readers, and read the tabs it last reported.
func NewPluginWindow(_ *State, _ noteport.NoteQueries, _ vaultport.VaultReaders) pluginport.PluginWindow {
	return pluginWindow{}
}

func (pluginWindow) FocusNote(context.Context, string) (domain.NoteRef, error) {
	return domain.NoteRef{}, errPluginWindowNotImplemented
}

func (pluginWindow) FocusPassages(context.Context, string, []domain.ByteRange) (string, error) {
	return "", errPluginWindowNotImplemented
}

func (pluginWindow) ReadOpenTabs(context.Context) (domain.OpenTabs, error) {
	return domain.OpenTabs{}, errPluginWindowNotImplemented
}
