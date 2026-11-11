package editor

import (
	"net/http"
	"testing"

	"connectrpc.com/connect"

	windowagent "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/command"
	filewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/file"
	mediawindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/media"
	notewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/note"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/search"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/settings"
	windowtext "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/text"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/plugin/catalog/when"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Services is the handlers a window answers through, one for each service of the schema, for the
// tests that call a handler without the HTTP stack, or put a window together from the handlers they
// choose.
type Services struct {
	state *State

	File    *filewindow.Handler
	Note    *notewindow.Handler
	Command *command.Handler
	Media   *mediawindow.Handler

	Themes numenv1connect.ThemeServiceHandler

	PluginServices PluginServices

	Plugin numenv1connect.PluginServiceHandler

	Extra []Route

	getPluginDir func(id string) (string, bool)

	server *FileServer

	readers     vaultport.VaultReaders
	pluginFiles *file.OpenPluginFile
	mediaTypes  *file.GetMediaType

	resolve *file.ResolveAddress

	NoteCases NoteUseCases
}

// NoteUseCases are the use cases a test gives the notes service. A use case left nil is one the
// test does not call.
type NoteUseCases struct {
	Read         *note.Read
	Write        *note.Write
	Create       *note.Create
	Rename       *note.Rename
	EditLinks    *note.EditLinks
	Names        edit.NameQueries
	OpeningNote  *query.GetOpeningNote
	Headings     *query.ListHeadings
	Neighborhood *query.GetNeighborhoodView
	ResolveLinks *query.ResolveLinks
}

// UseNotes builds the notes service over NoteCases.
func (s *Services) UseNotes() {
	use := s.NoteCases
	writes := s.state.Writes
	s.Note = notewindow.NewHandler(
		s.state.Notices,
		notewindow.NewProse(s.state, use.Read, use.Write, use.Create, use.Rename, writes),
		notewindow.NewOutline(s.state, use.OpeningNote, use.Neighborhood, use.Headings),
		notewindow.NewLinks(s.state, use.ResolveLinks, writeLink(use.EditLinks, use.Names, writes), nil, nil, writes),
		notewindow.NewConnections(s.state, nil, nil, nil, nil, nil),
	)
}

func createServices(state *State) *Services {
	services := &Services{
		state: state,
		File: filewindow.NewHandler(
			filewindow.NewFolders(state, state.Events, nil, nil, nil),
			filewindow.NewChanges(state, state.Notices, nil, nil, nil, nil),
		),
		Command: newEmptyCommandHandler(state),
		Media:   mediawindow.NewHandler(state, state, nil),
		Plugin:  numenv1connect.UnimplementedPluginServiceHandler{},
		server:  NewFileServer(state, nil),
	}
	services.Extra = []Route{
		RouteOf(state, search.NewHandler(state, nil, nil), search.NewRoute),
		RouteOf(state, settings.NewHandler(settings.Values{}, settings.PluginFolders{}, &settings.SecretItems{}, &settings.SecretGroups{}), settings.NewRoute),
		RouteOf(state, &windowagent.Handler{}, windowagent.NewRoute),
		RouteOf(state, windowtext.NewHandler(state, nil), windowtext.NewRoute),
	}
	services.UseNotes()
	return services
}

// NewHandler is the handler of the pages with these services in front of them.
func (s *Services) NewHandler(files http.Handler, named ...string) http.Handler {
	return s.pages().newHandler(files, named...)
}

func (s *Services) pages() pageSet {
	counted := s.state.Gate.CountQuestions()
	var routes []Route
	path, served := numenv1connect.NewFileServiceHandler(s.File, counted)
	routes = append(routes, NewRoute(path, served, s.File))
	path, served = numenv1connect.NewNoteServiceHandler(s.Note, counted)
	routes = append(routes, NewRoute(path, served, s.Note))
	path, served = numenv1connect.NewWindowServiceHandler(s.state.Window, counted)
	routes = append(routes, NewRoute(path, served, s.state.Window))
	path, served = numenv1connect.NewMediaServiceHandler(s.Media, counted)
	routes = append(routes, NewRoute(path, served, s.Media))
	path, served = numenv1connect.NewCommandServiceHandler(s.Command, counted)
	routes = append(routes, NewRoute(path, served, s.Command))
	path, served = numenv1connect.NewPluginServiceHandler(s.Plugin, counted)
	routes = append(routes, NewRoute(path, served, s.Plugin))
	routes = append(routes, s.Extra...)
	return pageSet{gate: s.state.Gate, window: statePages{state: s.state}, routes: routes, themes: s.Themes, plugins: s.PluginServices, pluginDir: s.getPluginDir}
}

// RouteOf is the route of a handler built by its own package: the path it serves and the handler
// that serves it come from build, counted the way the container counts them.
func RouteOf[H any](state *State, handler H, build func(H, ...connect.HandlerOption) (string, http.Handler)) Route {
	path, served := build(handler, state.Gate.CountQuestions())
	return NewRoute(path, served, handler)
}

// Mount puts the route in the place of the one served at its path, or beside them.
func (s *Services) Mount(route Route) {
	for i, one := range s.Extra {
		if one.path == route.path {
			s.Extra[i] = route
			return
		}
	}
	s.Extra = append(s.Extra, route)
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

func (s *Services) openPlaying(playing *Loopback) { s.state.Playing.Store(playing) }

func (s *Services) readFiles(readers vaultport.VaultReaders, openPluginFile *file.OpenPluginFile, mediaTypes *file.GetMediaType) {
	s.readers, s.pluginFiles, s.mediaTypes = readers, openPluginFile, mediaTypes
	s.findVaults(nil)
	s.resolve = file.NewResolveAddress(file.NewStatFile(readers), openPluginFile, mediaTypes)
	s.Media = mediawindow.NewHandler(s.state, s.state, s.resolve)
}

func (s *Services) findVaults(find file.VaultFinder) {
	s.server.open = file.NewOpenVaultFile(find, file.NewOpenFile(s.readers), s.pluginFiles, s.mediaTypes)
}

// ServicesOf is the handlers a window answers through.
func ServicesOf(o *Installation, state *State) *Services {
	services := &Services{state: state}
	for _, one := range o.pages.routes {
		switch handler := one.service.(type) {
		case *filewindow.Handler:
			services.File = handler
		case *notewindow.Handler:
			services.Note = handler
		case *mediawindow.Handler:
			services.Media = handler
		}
	}
	return services
}

// ViewOf is the view of the window as the core asks it to show something.
func ViewOf(state *State) windowsession.View {
	notices := state.Notices
	return windowsession.NewView(&notices.Places, &notices.Edits, &notices.ChangeEvents)
}

func writeLink(links *note.EditLinks, names edit.NameQueries, writes *wire.Writes) *note.WriteLink {
	if links == nil {
		return nil
	}
	linking := note.NewWriteLink(*links, names, writes)
	return &linking
}

// WrapPluginSwitch replaces the switch the window opens a vault through with the one wrap returns.
func WrapPluginSwitch(tb testing.TB, state *State, wrap func(next windowsession.VaultSwitch) windowsession.VaultSwitch) {
	tb.Helper()
	state.Switch = wrap(state.Switch)
}

func newEmptyCommandHandler(state *State) *command.Handler {
	list := plugincommand.NewList(nil, when.NewConditions())
	run := plugincommand.NewRun(nil, nil, nil, nil, nil, when.NewConditions())
	return command.NewHandler(state, plugincommand.NewListCommands(list, nil), plugincommand.NewRunCommand(list, run, nil), plugincommand.NewListItems(list, nil))
}
