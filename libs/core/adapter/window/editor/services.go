package editor

import (
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	setting "github.com/jiva-studio/numen/libs/core/usecase/settings"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/command"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/search"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/settings"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/textmenu"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// Services is the handlers a window answers through, one for each service of the
// schema, and the pieces beside them that are not a service.
//
// A handler holds the use cases it calls and the State it reads the vault
// from. The composition fills each in once the use cases are built.
type Services struct {
	state *State

	Vault     *vaultHandler
	Workspace *workspaceHandler
	File      *fileHandler
	Note      *noteHandler
	Search    *search.Handler
	Vaults    *vaultsHandler
	Settings  *settings.Handler
	Agent     *wire.AgentHandler
	Command   *command.Handler
	Recording *recordingHandler
	// TextVersion answers the menu a file's text versions make.
	TextVersion *textmenu.Handler

	// Themes are the stylesheets the window may be dressed in. They belong to
	// the installation, so they arrive here from whatever put the window
	// together. A build put together without a catalogue serves no ThemeService.
	Themes numenv1connect.ThemeServiceHandler

	// PluginServices answers the calls for the services plugins serve to the windows.
	PluginServices PluginServices

	// Plugin answers questions about the plugins installed in the application.
	Plugin numenv1connect.PluginServiceHandler

	// openChoosePath makes the use case that puts the picker for a plugin's path setting in front of the
	// person. Nil is a build that serves none.
	openChoosePath func(dialog machineport.PathDialog) *setting.ChoosePluginPath

	// getPluginDir returns the root directory of an installed plugin by ID, or false if not found.
	getPluginDir func(id string) (string, bool)

	// Playing is the socket a recording is played from. A build without one
	// answers with no address, and the window says the recording cannot be
	// played here.
	Playing *Loopback

	// UpdateEndpoint is where the release manifest is asked for. Empty asks the published one.
	UpdateEndpoint string

	server *fileServer
}

// createServices is every handler of a window over this state, each holding
// nothing yet that needs a vault.
func createServices(state *State) *Services {
	return &Services{
		state:       state,
		Vault:       &vaultHandler{state: state},
		Workspace:   &workspaceHandler{state: state},
		File:        &fileHandler{state: state},
		Note:        &noteHandler{state: state},
		Search:      search.NewHandler(state),
		Vaults:      &vaultsHandler{state: state},
		Settings:    settings.NewHandler(),
		Agent:       newAgentHandler(state),
		Command:     command.NewHandler(plugincommand.NewCommands(nil, nil)),
		Recording:   &recordingHandler{state: state},
		TextVersion: textmenu.NewHandler(state, nil),
		Plugin:      numenv1connect.UnimplementedPluginServiceHandler{},
		server:      &fileServer{state: state},
	}
}

// openPlaying names the socket a recording is played from.
func (s *Services) openPlaying(playing *Loopback) {
	s.Playing = playing
	s.Recording.playing = playing
}

// readFiles binds the use cases that read the files of the vaults the readers open, and the copies
// played in place of files.
func (s *Services) readFiles(readers vaultport.VaultReaders, openCopy *file.OpenCopy) {
	s.server.openFile = file.NewOpenFile(readers)
	s.server.openCopy = openCopy
	s.Recording.getFingerprint = file.NewGetFingerprint(readers)
	s.Recording.openCopy = openCopy
}

// noteHandler answers what a vault's notes hold and what changes them.
type noteHandler struct {
	state *State

	getOpeningNote *note.GetOpeningNote
	getTypes       *note.GetTypes
	getHeadings    *note.GetHeadings
	resolveLinks   *note.ResolveLinks

	read              *note.Read
	write             *note.Write
	create            *note.Create
	editLinks         *note.EditLinks
	replaceLinks      *note.ReplaceLinks
	rename            *note.Rename
	showNeighbourhood *note.ShowNeighbourhood
	getLinkAddress    *note.GetLinkAddress
}

// SetFolderDialog puts the machine's file and folder dialogs in front of the person when a
// client asks for one.
func (s *Services) SetFolderDialog(dialog machineport.PathDialog) {
	choose := vaults.NewChooseFolder(dialog)
	s.Vaults.chooseFolder = &choose
	if s.openChoosePath != nil && s.Settings != nil {
		s.Settings.ChoosePath = s.openChoosePath(dialog)
	}
}

// GetPickFolder is the use case that lets an agent put the same dialog in front
// of the person, or nil where the window has none.
func (s *Services) GetPickFolder() *vaults.PickFolder {
	if s.Vaults.chooseFolder == nil {
		return nil
	}
	pick := s.Vaults.chooseFolder.GetPickFolder()
	return &pick
}

// newAgentHandler answers for the agent the panel's tasks go to. A task naming no folder is started
// in the vault open in the window.
func newAgentHandler(state *State) *wire.AgentHandler {
	return &wire.AgentHandler{
		GetRoot:        func() string { return state.GetCurrentVault().Path },
		GetUnreachable: state.Unreachable.Why,
	}
}
