package editor

import (
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	setting "github.com/jiva-studio/numen/libs/core/usecase/settings"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/command"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/search"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/settings"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/textmenu"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/texts"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	machineport "github.com/jiva-studio/numen/libs/core/usecase/file/machinecontract"
	vaultport "github.com/jiva-studio/numen/libs/core/usecase/vault/vaultcontract"
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
	Media     *mediaHandler
	// TextVersion answers the menu a file's text versions make.
	TextVersion *textmenu.Handler
	// Text serves the live text of a file and its edits to the views.
	Text *texts.Handler

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

	// Playing is the socket that serves recording playback. A build without one
	// returns no address, and the window reports that playback is unavailable.
	Playing *Loopback

	// UpdateEndpoint is the URL of the release manifest. When empty, the published URL is used.
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
		Command:     command.NewHandler(state, plugincommand.NewCommands(nil, nil, nil)),
		Media:       &mediaHandler{state: state},
		TextVersion: textmenu.NewHandler(state, nil),
		Text:        texts.NewHandler(state, nil),
		Plugin:      numenv1connect.UnimplementedPluginServiceHandler{},
		server:      &fileServer{state: state},
	}
}

// openPlaying names the socket a recording is played from.
func (s *Services) openPlaying(playing *Loopback) {
	s.Playing = playing
	s.Media.playing = playing
}

// readFiles binds the use cases that read the files of the vaults the readers open, the copies
// played in place of files, and the content types files are served as.
func (s *Services) readFiles(readers vaultport.VaultReaders, openCopy *file.OpenProxy, mediaTypes *file.GetMediaType) {
	s.server.openFile = file.NewOpenFile(readers)
	s.server.openCopy = openCopy
	s.server.mediaTypes = mediaTypes
	s.Media.statFile = file.NewStatFile(readers)
	s.Media.openCopy = openCopy
	s.Media.mediaTypes = mediaTypes
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

	getConnection    *note.GetConnection
	saveConnection   *note.SaveConnection
	removeSide       *note.RemoveSide
	removeConnection *note.RemoveConnection
	join             *note.Join
	createInRelation *note.CreateInRelation
	getBacklinks     *note.GetBacklinks
}

// SetFolderDialog sets the native file and folder dialog used when a client
// requests one.
func (s *Services) SetFolderDialog(dialog machineport.PathDialog) {
	choose := vaults.NewChooseFolder(dialog)
	s.Vaults.chooseFolder = &choose
	if s.openChoosePath != nil && s.Settings != nil {
		s.Settings.ChoosePath = s.openChoosePath(dialog)
	}
}

// GetPickFolder returns the use case that lets an agent open the same dialog,
// or nil when the window has no dialog.
func (s *Services) GetPickFolder() *vaults.PickFolder {
	if s.Vaults.chooseFolder == nil {
		return nil
	}
	pick := s.Vaults.chooseFolder.GetPickFolder()
	return &pick
}

// newAgentHandler answers for the agent the panel's tasks go to. A task naming no folder is started
// in the vault open in the window, and works it.
func newAgentHandler(state *State) *wire.AgentHandler {
	return &wire.AgentHandler{
		GetRoot:        func() string { return state.GetCurrentVault().Path },
		GetVault:       func() domain.VaultID { return state.GetCurrentVault().ID },
		GetUnreachable: state.Unreachable.Why,
	}
}
