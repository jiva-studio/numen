package editor

import (
	"context"
	"embed"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	filewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/file"
	mediawindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/media"
	notewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/note"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/domain"
)

//go:embed all:pages
var pages embed.FS

// errShut is returned for a request to a window that has been shut; the window
// answers no further requests.
var errShut = wire.ErrShut

// Pages is the interface itself, built by `make interface` and embedded in the
// binary. A binary built without it reports an error.
func Pages() (http.Handler, error) { return wire.NewInterfaceServer(pages) }

// PluginServices is where the calls for the services plugins serve are forwarded.
type PluginServices interface {
	http.Handler
	// Serves reports whether an enabled plugin serves the service the path names.
	Serves(urlPath string) bool
	// ServeRoute forwards a request on the plugin route, /plugin/<id>/..., to that plugin.
	ServeRoute(w http.ResponseWriter, r *http.Request)
	// ListFrameOrigins lists the origins the pages of every enabled plugin frame.
	ListFrameOrigins() []string
	// GetFrameOrigins lists the origins the pages of one enabled plugin frame.
	GetFrameOrigins(pluginID string) []string
}

// Route is one service this build answers: where its calls arrive, and what
// answers them.
type Route struct {
	path    string
	handler http.Handler
	service any
}

// NewRoute is the route of a service of the schema. Handler is what the schema
// generated for the service, and service is the handler it answers through.
func NewRoute(path string, handler http.Handler, service any) Route {
	return Route{path: path, handler: handler, service: service}
}

// pageSet is what answers the requests that reach a window: the routes of its
// services and the services beside them.
type pageSet struct {
	gate      *wire.Gate
	window    pageWindow
	routes    []Route
	themes    numenv1connect.ThemeServiceHandler
	plugins   PluginServices
	pluginDir func(id string) (string, bool)
}

// pageWindow is what the pages and the installation need of the window besides its gate: the vault it
// shows, where its work is listed, where recordings are played from, and the wait for its pages to write.
type pageWindow interface {
	shownVault
	taskList
	getPlayOrigins() []string
	SetPlaying(playing *Loopback)
	RemoveTask(id string)
	WaitForFlush(ctx context.Context) bool
}

// statePages is the window of a State, as its pages see it.
type statePages struct {
	state *State
}

func (s statePages) GetCurrentVault() domain.Vault { return s.state.GetCurrentVault() }

func (s statePages) AddTask(pos work.Task) { s.state.Window.AddTask(pos) }

func (s statePages) RemoveTask(id string) { s.state.Window.RemoveTask(id) }

func (s statePages) WaitForFlush(ctx context.Context) bool { return s.state.Window.WaitForFlush(ctx) }

func (s statePages) SetPlaying(playing *Loopback) { s.state.Playing.Store(playing) }

func (s statePages) getPlayOrigins() []string { return s.state.Playing.Load().getPlayOrigins() }

// NewHandler puts the services in front of the pages, so a window and a browser
// are served by one handler.
// named lists the services this build serves, by the names the schema gives
// them. A service left out is not mounted, so a call to it is unanswered.
// Naming none builds the whole schema.
func (o *Installation) NewHandler(files http.Handler, named ...string) http.Handler {
	return o.pages.newHandler(files, named...)
}

func (s pageSet) newHandler(files http.Handler, named ...string) http.Handler {
	routes := s.newRoutes(named)
	// Only a build that answers the whole schema serves the bytes of vault files and the plugin route.
	isWhole := len(named) == 0

	// The origin recordings play from is known once the socket it is served over is
	// open, which precedes any page request.
	played := s.window.getPlayOrigins()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", s.getPolicy(played))
		// A window being taken away answers nothing.
		if s.gate.IsClosed() {
			http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
			return
		}
		// A service served by an enabled plugin takes precedence in a build that answers the whole schema.
		isAgent := strings.HasPrefix(r.URL.Path, "/numen.v1.AgentService/")
		if !isAgent && isWhole && s.plugins != nil && s.plugins.Serves(r.URL.Path) {
			s.setRequestVault(r)
			s.plugins.ServeHTTP(w, r)
			return
		}
		for _, one := range routes {
			if strings.HasPrefix(r.URL.Path, one.path) {
				one.handler.ServeHTTP(w, r)
				return
			}
		}
		s.serveAdjacent(w, r, isWhole, files)
	})
}

func isServed(named []string, route Route) bool {
	return len(named) == 0 || slices.Contains(named, strings.Trim(route.path, "/"))
}

func (s pageSet) newRoutes(named []string) []Route {
	var routes []Route
	for _, one := range s.routes {
		if isServed(named, one) {
			routes = append(routes, one)
		}
	}
	// The themes belong to the installation and arrive here from whatever put
	// the window together, so a build put together without a catalog serves
	// none.
	if s.themes != nil {
		path, handler := numenv1connect.NewThemeServiceHandler(s.themes, s.gate.CountQuestions())
		if themed := NewRoute(path, handler, s.themes); isServed(named, themed) {
			routes = append(routes, themed)
		}
	}
	return routes
}

func (s pageSet) serveAdjacent(w http.ResponseWriter, r *http.Request, isWhole bool, files http.Handler) {
	switch {
	case strings.HasPrefix(r.URL.Path, wire.PluginRoute):
		if !s.gate.BeginQuestion() {
			http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
			return
		}
		defer s.gate.FinishQuestion()
		s.servePluginRoute(w, r, isWhole)
	case isWhole && strings.HasPrefix(r.URL.Path, assetsRoute):
		if !s.gate.BeginQuestion() {
			http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
			return
		}
		defer s.gate.FinishQuestion()
		s.serveAsset(w, r)
	case slices.Contains(wire.GetOpenedAt(), r.URL.Path):
		wire.WritePage(w, r, pages, s.themes, files)
	default:
		files.ServeHTTP(w, r)
	}
}

// assetsRoute is the path prefix where the bytes of vault files are served.
const assetsRoute = "/assets/"

func (s pageSet) serveAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rest := strings.TrimPrefix(r.URL.EscapedPath(), assetsRoute)
	unescapedPath, err := url.PathUnescape(rest)
	if err != nil || unescapedPath == "" {
		http.Error(w, "invalid asset path", http.StatusBadRequest)
		return
	}
	if s.serveVaultAsset(w, r, unescapedPath) {
		return
	}
	http.Error(w, "asset not found", http.StatusNotFound)
}

func (s pageSet) serveVaultAsset(w http.ResponseWriter, r *http.Request, unescapedPath string) bool {
	vault := s.window.GetCurrentVault()
	name := filepath.FromSlash(unescapedPath)
	if vault.Path == "" || !filepath.IsLocal(name) {
		return false
	}
	file, info, err := openVaultFile(vault.Path, name)
	if err != nil {
		return false
	}
	defer closeAndReport(s.window.AddTask, file)
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	return true
}

func openVaultFile(vaultDir, name string) (*os.File, os.FileInfo, error) {
	root, err := os.OpenRoot(vaultDir)
	if err != nil {
		return nil, nil, err
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, nil, errors.Join(err, root.Close())
	}
	if err := root.Close(); err != nil {
		return nil, nil, errors.Join(err, file.Close())
	}
	info, err := file.Stat()
	if err == nil && info.IsDir() {
		err = os.ErrNotExist
	}
	if err != nil {
		return nil, nil, errors.Join(err, file.Close())
	}
	return file, info, nil
}

func (s pageSet) servePluginRoute(w http.ResponseWriter, r *http.Request, isWhole bool) {
	if wire.ServePluginFile(w, r, s.pluginDir) {
		return
	}
	if !isWhole || s.plugins == nil {
		http.Error(w, "plugin file not found", http.StatusNotFound)
		return
	}
	s.setRequestVault(r)
	s.plugins.ServeRoute(w, r)
}

// requestVaultHeader names the vault a request forwarded to a plugin works, as the host endpoint
// reads it on the calls the plugin makes back.
const requestVaultHeader = "Numen-Vault"

func (s pageSet) setRequestVault(r *http.Request) {
	if r.Header.Get(requestVaultHeader) != "" {
		return
	}
	if v := s.window.GetCurrentVault(); v.ID != "" {
		r.Header.Set(requestVaultHeader, string(v.ID))
	}
}

// NewNoteRoute is the route of the notes service.
func NewNoteRoute(state *State, handler *notewindow.Handler) Route {
	path, served := numenv1connect.NewNoteServiceHandler(handler, state.Gate.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewFileRoute is the route of the files service.
func NewFileRoute(state *State, handler *filewindow.Handler) Route {
	path, served := numenv1connect.NewFileServiceHandler(handler, state.Gate.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewMediaRoute is the route of the service that tells where a media file is played from.
func NewMediaRoute(state *State, handler *mediawindow.Handler) Route {
	path, served := numenv1connect.NewMediaServiceHandler(handler, state.Gate.CountQuestions())
	return NewRoute(path, served, handler)
}
