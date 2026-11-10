package editor

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/command"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/search"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/texts"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tabkind"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Services is the handlers a window answers through, one for each service of the schema, for the
// tests that call a handler without the HTTP stack, or put a window together from the handlers they
// choose.
type Services struct {
	state *State

	Vault     *vaultHandler
	Workspace *workspaceHandler
	File      *FileHandler
	Note      *NoteHandler
	Search    *search.Handler
	Vaults    *VaultsHandler
	Settings  *settings.Handler
	Agent     *wire.AgentHandler
	Command   *command.Handler
	Media     *MediaHandler
	Text      *texts.Handler

	Themes numenv1connect.ThemeServiceHandler

	PluginServices PluginServices

	Plugin numenv1connect.PluginServiceHandler

	Catalog numenv1connect.CatalogServiceHandler

	getPluginDir func(id string) (string, bool)

	server *FileServer
}

func createServices(state *State) *Services {
	return &Services{
		state:     state,
		Vault:     &vaultHandler{state: state},
		Workspace: &workspaceHandler{state: state},
		File:      NewFileHandler(state),
		Note:      NewNoteHandler(state),
		Search:    search.NewHandler(state),
		Vaults:    NewVaultsHandler(state),
		Settings:  settings.NewHandler(),
		Agent:     newAgentHandler(state),
		Command:   command.NewHandler(state, plugincommand.NewCommands(nil, nil, nil, nil), nil),
		Media:     NewMediaHandler(state, nil, nil),
		Text:      texts.NewHandler(state, nil),
		Plugin:    numenv1connect.UnimplementedPluginServiceHandler{},
		server:    NewFileServer(state, nil, nil, nil, nil),
	}
}

// NewHandler is the handler of the pages with these services in front of them.
func (s *Services) NewHandler(files http.Handler, named ...string) http.Handler {
	return s.pages().newHandler(files, named...)
}

func (s *Services) pages() pageSet {
	counted := s.state.CountQuestions()
	var routes []Route
	path, served := numenv1connect.NewVaultServiceHandler(s.Vault, counted)
	routes = append(routes, NewRoute(path, served, s.Vault))
	path, served = numenv1connect.NewWorkspaceServiceHandler(s.Workspace, counted)
	routes = append(routes, NewRoute(path, served, s.Workspace))
	path, served = numenv1connect.NewFileServiceHandler(s.File, counted)
	routes = append(routes, NewRoute(path, served, s.File))
	path, served = numenv1connect.NewNoteServiceHandler(s.Note, counted)
	routes = append(routes, NewRoute(path, served, s.Note))
	path, served = numenv1connect.NewSearchServiceHandler(s.Search, counted)
	routes = append(routes, NewRoute(path, served, s.Search))
	path, served = numenv1connect.NewVaultsServiceHandler(s.Vaults, counted)
	routes = append(routes, NewRoute(path, served, s.Vaults))
	path, served = numenv1connect.NewSettingsServiceHandler(s.Settings, counted)
	routes = append(routes, NewRoute(path, served, s.Settings))
	path, served = numenv1connect.NewWindowServiceHandler(s.state.Window, counted)
	routes = append(routes, NewRoute(path, served, s.state.Window))
	path, served = numenv1connect.NewMediaServiceHandler(s.Media, counted)
	routes = append(routes, NewRoute(path, served, s.Media))
	path, served = numenv1connect.NewAgentServiceHandler(s.Agent, counted)
	routes = append(routes, NewRoute(path, served, s.Agent))
	path, served = numenv1connect.NewCommandServiceHandler(s.Command, counted)
	routes = append(routes, NewRoute(path, served, s.Command))
	path, served = numenv1connect.NewTextServiceHandler(s.Text, counted)
	routes = append(routes, NewRoute(path, served, s.Text))
	path, served = numenv1connect.NewPluginServiceHandler(s.Plugin, counted)
	routes = append(routes, NewRoute(path, served, s.Plugin))
	if s.Catalog != nil {
		path, served = numenv1connect.NewCatalogServiceHandler(s.Catalog, counted)
		routes = append(routes, NewRoute(path, served, s.Catalog))
	}
	return pageSet{state: s.state, routes: routes, themes: s.Themes, plugins: s.PluginServices, pluginDir: s.getPluginDir}
}

func (s *Services) openPlaying(playing *Loopback) { s.state.Playing.Store(playing) }

func (s *Services) readFiles(readers vaultport.VaultReaders, openPluginFile *file.OpenPluginFile, mediaTypes *file.GetMediaType) {
	s.server.openFile = file.NewOpenFile(readers)
	s.server.openPluginFile = openPluginFile
	s.server.mediaTypes = mediaTypes
	s.Media.resolveAddress = file.NewResolveAddress(file.NewStatFile(readers), openPluginFile)
	s.Media.mediaTypes = mediaTypes
}

// ServicesOf is the handlers a window answers through.
func ServicesOf(o *Installation) *Services {
	services := &Services{state: o.State}
	for _, one := range o.pages.routes {
		switch handler := one.service.(type) {
		case *vaultHandler:
			services.Vault = handler
		case *workspaceHandler:
			services.Workspace = handler
		case *FileHandler:
			services.File = handler
		case *NoteHandler:
			services.Note = handler
		case *search.Handler:
			services.Search = handler
		case *VaultsHandler:
			services.Vaults = handler
		case *settings.Handler:
			services.Settings = handler
		case *MediaHandler:
			services.Media = handler
		}
	}
	return services
}

// The use cases a handler holds, for the tests that wrap one in a fake.

func (s *Services) NoteCreate() *note.Create { return s.Note.create }
func (s *Services) NoteWrite() *note.Write   { return s.Note.write }
func (s *Services) NoteRename() *note.Rename { return s.Note.rename }
func (s *Services) FileRemove() *note.Remove { return s.File.remove }
func (s *Services) FileMove() *vaults.Move   { return s.File.move }

func (s *Services) SetFileMove(move *vaults.Move)  { s.File.move = move }
func (s *Services) SetNoteWrite(write *note.Write) { s.Note.write = write }

func (s *Services) SetTabKindChoice(choose *tabkind.Choose) { s.File.chooseTabKind = choose }

// NewWireModels is the models as the wire carries them.
func NewWireModels(target []settingsdomain.Model) []*v1.Model { return settings.NewWireModels(target) }
