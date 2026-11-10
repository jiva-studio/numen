package editor

import (
	"net/http"

	"connectrpc.com/connect"

	windowagent "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/command"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/search"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/settings"
	windowtext "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/text"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tabkind"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
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
	Vaults    *VaultsHandler
	Command   *command.Handler
	Media     *MediaHandler

	Themes numenv1connect.ThemeServiceHandler

	PluginServices PluginServices

	Plugin numenv1connect.PluginServiceHandler

	Mounted []Route

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
		Vaults:    NewVaultsHandler(state),
		Command:   command.NewHandler(state, plugincommand.NewCommands(nil, nil, nil, nil), nil),
		Media:     NewMediaHandler(state, nil, nil),
		Plugin:    numenv1connect.UnimplementedPluginServiceHandler{},
		server:    NewFileServer(state, nil, nil, nil, nil),
		Mounted: []Route{
			RouteOf(state, search.NewHandler(state), search.NewRoute),
			RouteOf(state, &settings.Handler{}, settings.NewRoute),
			RouteOf(state, &windowagent.Handler{}, windowagent.NewRoute),
			RouteOf(state, windowtext.NewHandler(state, nil), windowtext.NewRoute),
		},
	}
}

// RouteOf is the route of a handler built by its own package: the path it serves and the handler
// that serves it come from build, counted the way the container counts them.
func RouteOf[H any](state *State, handler H, build func(H, ...connect.HandlerOption) (string, http.Handler)) Route {
	path, served := build(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// Mount puts the route in the place of the one served at its path, or beside them.
func (s *Services) Mount(route Route) {
	for i, one := range s.Mounted {
		if one.path == route.path {
			s.Mounted[i] = route
			return
		}
	}
	s.Mounted = append(s.Mounted, route)
}

// HandlerOf is the handler of the type the installation mounted.
func HandlerOf[T any](o *Installation) T {
	for _, one := range o.pages.routes {
		if handler, ok := one.service.(T); ok {
			return handler
		}
	}
	var none T
	return none
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
	path, served = numenv1connect.NewVaultsServiceHandler(s.Vaults, counted)
	routes = append(routes, NewRoute(path, served, s.Vaults))
	path, served = numenv1connect.NewWindowServiceHandler(s.state.Window, counted)
	routes = append(routes, NewRoute(path, served, s.state.Window))
	path, served = numenv1connect.NewMediaServiceHandler(s.Media, counted)
	routes = append(routes, NewRoute(path, served, s.Media))
	path, served = numenv1connect.NewCommandServiceHandler(s.Command, counted)
	routes = append(routes, NewRoute(path, served, s.Command))
	path, served = numenv1connect.NewPluginServiceHandler(s.Plugin, counted)
	routes = append(routes, NewRoute(path, served, s.Plugin))
	routes = append(routes, s.Mounted...)
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
		case *VaultsHandler:
			services.Vaults = handler
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
