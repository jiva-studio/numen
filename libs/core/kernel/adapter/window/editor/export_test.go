package editor

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/command"
	filewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/file"
	mediawindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/media"
	notewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/note"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/search"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/texts"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
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
	File      *filewindow.Handler
	Note      *notewindow.Handler
	Search    *search.Handler
	Vaults    *VaultsHandler
	Settings  *settings.Handler
	Agent     *wire.AgentHandler
	Command   *command.Handler
	Media     *mediawindow.Handler
	Text      *texts.Handler

	Themes numenv1connect.ThemeServiceHandler

	PluginServices PluginServices

	Plugin numenv1connect.PluginServiceHandler

	Catalog numenv1connect.CatalogServiceHandler

	getPluginDir func(id string) (string, bool)

	server *FileServer

	installation *Installation

	resolve *file.ResolveAddress

	NoteCases NoteUseCases
}

// NoteUseCases are the use cases a test gives the notes service. A use case left nil is one the
// test does not call.
type NoteUseCases struct {
	Read           *note.Read
	Write          *note.Write
	Create         *note.Create
	Rename         *note.Rename
	EditLinks      *note.EditLinks
	GetLinkAddress *query.GetLinkAddress
	OpeningNote    *query.GetOpeningNote
	Types          *query.GetTypes
	Headings       *query.GetHeadings
	Neighbourhood  *query.ShowNeighbourhood
	ResolveLinks   *query.ResolveLinks
}

// UseNotes builds the notes service over NoteCases.
func (s *Services) UseNotes() {
	use := s.NoteCases
	addresses := notewindow.NewAddresses(s.state, use.GetLinkAddress)
	s.Note = notewindow.NewHandler(
		s.state,
		notewindow.NewProse(s.state, s.state, addresses, use.Read, use.Write, use.Create, use.Rename),
		notewindow.NewOutline(s.state, nil, use.OpeningNote, use.Types, use.Headings, use.Neighbourhood),
		notewindow.NewLinks(s.state, s.state, addresses, use.ResolveLinks, use.EditLinks, nil, nil),
		notewindow.NewConnections(s.state, s.state, nil, nil, nil, nil, nil),
	)
}

func createServices(state *State) *Services {
	services := &Services{
		state:     state,
		Vault:     &vaultHandler{state: state},
		Workspace: &workspaceHandler{state: state},
		File: filewindow.NewHandler(
			filewindow.NewFolders(state, state, state, nil, nil, nil, nil),
			filewindow.NewChanges(state, state, state, state, nil, nil, nil),
		),
		Search:   search.NewHandler(state),
		Vaults:   NewVaultsHandler(state),
		Settings: settings.NewHandler(),
		Agent:    newAgentHandler(state),
		Command:  command.NewHandler(state, plugincommand.NewCommands(nil, nil, nil, nil), nil),
		Media:    mediawindow.NewHandler(state, state, nil, nil),
		Text:     texts.NewHandler(state, nil),
		Plugin:   numenv1connect.UnimplementedPluginServiceHandler{},
		server:   NewFileServer(state, nil, nil, nil, nil),
	}
	services.UseNotes()
	return services
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
	s.resolve = file.NewResolveAddress(file.NewStatFile(readers), openPluginFile)
	s.Media = mediawindow.NewHandler(
		s.state, s.state, s.resolve, mediaTypes)
}

// ServicesOf is the handlers a window answers through.
func ServicesOf(o *Installation) *Services {
	services := &Services{state: o.State, installation: o}
	for _, one := range o.pages.routes {
		switch handler := one.service.(type) {
		case *vaultHandler:
			services.Vault = handler
		case *workspaceHandler:
			services.Workspace = handler
		case *filewindow.Handler:
			services.File = handler
		case *notewindow.Handler:
			services.Note = handler
		case *search.Handler:
			services.Search = handler
		case *VaultsHandler:
			services.Vaults = handler
		case *settings.Handler:
			services.Settings = handler
		case *mediawindow.Handler:
			services.Media = handler
		}
	}
	return services
}

// The use cases the services of notes and files call, for the tests that wrap one in a fake. The
// services hold pointers into the scenarios of the installation, so the one wrapped there is the
// one they call.

func (s *Services) NoteCreate() *note.Create { return &s.installation.notes.Create }
func (s *Services) NoteWrite() *note.Write   { return &s.installation.notes.Write }
func (s *Services) NoteRename() *note.Rename { return &s.installation.notes.Rename }
func (s *Services) FileRemove() *note.Remove { return &s.installation.notes.Remove }
func (s *Services) FileMove() *vaults.Move   { return &s.installation.vaults.Move }

func (s *Services) SetFileMove(move *vaults.Move)  { s.installation.vaults.Move = *move }
func (s *Services) SetNoteWrite(write *note.Write) { s.installation.notes.Write = *write }

// NewWireModels is the models as the wire carries them.
func NewWireModels(target []settingsdomain.Model) []*v1.Model { return settings.NewWireModels(target) }
