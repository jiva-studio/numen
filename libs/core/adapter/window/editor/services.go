package editor

import (
	"context"

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
	Text      *texts.Handler

	Themes numenv1connect.ThemeServiceHandler

	PluginServices PluginServices

	Plugin numenv1connect.PluginServiceHandler

	Connection numenv1connect.ProviderConnectionServiceHandler

	Catalog numenv1connect.CatalogServiceHandler

	openChoosePath func(dialog machineport.PathDialog) *setting.ChoosePluginPath

	getPluginDir func(id string) (string, bool)

	Playing *Loopback

	server *fileServer
}

func createServices(state *State) *Services {
	return &Services{
		state:      state,
		Vault:      &vaultHandler{state: state},
		Workspace:  &workspaceHandler{state: state},
		File:       &fileHandler{state: state},
		Note:       &noteHandler{state: state},
		Search:     search.NewHandler(state),
		Vaults:     &vaultsHandler{state: state},
		Settings:   settings.NewHandler(),
		Agent:      newAgentHandler(state),
		Command:    command.NewHandler(state, plugincommand.NewCommands(nil, nil, nil, nil), nil),
		Media:      &mediaHandler{state: state},
		Text:       texts.NewHandler(state, nil),
		Plugin:     numenv1connect.UnimplementedPluginServiceHandler{},
		Connection: numenv1connect.UnimplementedProviderConnectionServiceHandler{},
		server:     &fileServer{state: state},
	}
}

func (s *Services) openPlaying(playing *Loopback) {
	s.Playing = playing
	s.Media.playing = playing
}

func (s *Services) readFiles(readers vaultport.VaultReaders, openPluginFile *file.OpenPluginFile, mediaTypes *file.GetMediaType) {
	s.server.openFile = file.NewOpenFile(readers)
	s.server.openPluginFile = openPluginFile
	s.server.mediaTypes = mediaTypes
	s.Media.resolveAddress = file.NewResolveAddress(file.NewStatFile(readers), openPluginFile)
	s.Media.mediaTypes = mediaTypes
}

// noteHandler answers what a vault's notes hold and what changes them.
type noteHandler struct {
	state *State

	getOpeningNote *note.GetOpeningNote
	getTypes       *note.GetTypes
	findTabKinds   func(ctx context.Context, v domain.Vault, paths []string) (map[string]string, error)
	getHeadings    *note.GetHeadings
	resolveLinks   *note.ResolveLinks

	read              *note.Read
	write             *note.Write
	create            *note.Create
	editLinks         *note.EditLinks
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

func newAgentHandler(state *State) *wire.AgentHandler {
	return &wire.AgentHandler{
		GetRoot:        func() string { return state.GetCurrentVault().Path },
		GetVault:       func() domain.VaultID { return state.GetCurrentVault().ID },
		GetUnreachable: state.Unreachable.Why,
	}
}
