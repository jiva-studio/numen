package editor

import (
	"net/http"

	"connectrpc.com/connect"

	windowagent "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/command"
	filewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/file"
	mediawindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/media"
	notewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/note"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/search"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/settings"
	windowtext "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/text"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

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
	Neighborhood   *query.ShowNeighborhood
	ResolveLinks   *query.ResolveLinks
}

// UseNotes builds the notes service over NoteCases.
func (s *Services) UseNotes() {
	use := s.NoteCases
	session := NewSession(s.state)
	addresses := notewindow.NewAddresses(s.state, use.GetLinkAddress)
	s.Note = notewindow.NewHandler(
		s.state.Notices,
		notewindow.NewProse(session, addresses, use.Read, saveProse(use.Write, session), createNote(use.Create, session), use.Rename),
		notewindow.NewOutline(s.state, nil, use.OpeningNote, use.Types, use.Headings, use.Neighborhood),
		notewindow.NewLinks(session, addresses, use.ResolveLinks, use.EditLinks, nil, nil),
		notewindow.NewConnections(session, nil, nil, nil, nil, nil),
	)
}

func createServices(state *State) *Services {
	services := &Services{
		state: state,
		File: filewindow.NewHandler(
			filewindow.NewFolders(NewSession(state), state.Events, nil, nil, nil, nil),
			filewindow.NewChanges(NewSession(state), state.Writes, state.Notices, nil, nil, nil),
		),
		Command: command.NewHandler(state, plugincommand.NewCommands(nil, nil, nil, nil, nil), nil),
		Media:   mediawindow.NewHandler(state, state, nil, nil),
		Plugin:  numenv1connect.UnimplementedPluginServiceHandler{},
		server:  NewFileServer(state, nil, nil, nil, nil),
	}
	services.Extra = []Route{
		RouteOf(state, search.NewHandler(state), search.NewRoute),
		RouteOf(state, &settings.Handler{}, settings.NewRoute),
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
	return pageSet{state: s.state, routes: routes, themes: s.Themes, plugins: s.PluginServices, pluginDir: s.getPluginDir}
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

// The use cases the services of notes and files call, for the tests that wrap one in a fake. The
// services hold pointers into the scenarios of the installation, so the one wrapped there is the
// one they call.

func (s *Services) NoteCreate() *note.Create { return &s.installation.scenarios.Notes.Edit.Create }
func (s *Services) NoteWrite() *note.Write   { return &s.installation.scenarios.Notes.Edit.Write }
func (s *Services) NoteRename() *note.Rename { return &s.installation.scenarios.Notes.Place.Rename }
func (s *Services) FileRemove() *note.Remove { return &s.installation.scenarios.Notes.Place.Remove }
func (s *Services) FileMove() *vaults.Move   { return &s.installation.scenarios.Vaults.Tree.Move }

func (s *Services) SetFileMove(move *vaults.Move) { s.installation.scenarios.Vaults.Tree.Move = *move }
func (s *Services) SetNoteWrite(write *note.Write) {
	s.installation.scenarios.Notes.Edit.Write = *write
}

// ViewOf is the view of the window as the core asks it to show something.
func ViewOf(state *State) windowsession.View {
	notices := state.Notices
	return windowsession.NewView(&notices.Places, &notices.Edits, &notices.ChangeEvents)
}

func saveProse(write *note.Write, session Session) *note.SaveProse {
	if write == nil {
		return nil
	}
	saved := note.NewSaveProse(write, session, session)
	return &saved
}

func createNote(create *note.Create, session Session) *note.CreateNote {
	if create == nil {
		return nil
	}
	made := note.NewCreateNote(create, session, session)
	return &made
}
