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

	"connectrpc.com/connect"

	filewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/file"
	mediawindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/media"
	notewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/note"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/plugincatalog"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/search"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/texts"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

//go:embed all:pages
var pages embed.FS

// errShut is returned for a request to a window that has been shut; the window
// answers no further requests.
var errShut = errors.New("this window is shut")

// Pages is the interface itself, built by `make interface` and embedded in the
// binary. A binary built without it reports an error.
func Pages() (http.Handler, error) { return wire.NewInterfaceServer(pages) }

// PluginServices is where the calls for the services plugins serve are forwarded.
type PluginServices = interface {
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
	state     *State
	routes    []Route
	themes    numenv1connect.ThemeServiceHandler
	plugins   PluginServices
	pluginDir func(id string) (string, bool)
}

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
	played := s.state.Playing.Load().getPlayOrigins()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", s.getPolicy(played))
		// A window being taken away answers nothing.
		if s.state.isClosed() {
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
		path, handler := numenv1connect.NewThemeServiceHandler(s.themes, s.state.CountQuestions())
		if themed := NewRoute(path, handler, s.themes); isServed(named, themed) {
			routes = append(routes, themed)
		}
	}
	return routes
}

func (s pageSet) serveAdjacent(w http.ResponseWriter, r *http.Request, isWhole bool, files http.Handler) {
	switch {
	case strings.HasPrefix(r.URL.Path, wire.PluginRoute):
		if !s.state.questions.begin() {
			http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
			return
		}
		defer s.state.questions.finish()
		s.servePluginRoute(w, r, isWhole)
	case isWhole && strings.HasPrefix(r.URL.Path, assetsRoute):
		if !s.state.questions.begin() {
			http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
			return
		}
		defer s.state.questions.finish()
		s.serveAsset(w, r)
	case slices.Contains(wire.GetOpenedAt(), r.URL.Path):
		wire.WritePage(w, r, pages, s.themes, files)
	default:
		files.ServeHTTP(w, r)
	}
}

// CountQuestions counts each call to a service among the questions the window
// is answering, and refuses it once the window is shut.
func (s *State) CountQuestions() connect.HandlerOption {
	return connect.WithInterceptors(connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, r connect.AnyRequest) (connect.AnyResponse, error) {
				if !s.questions.begin() {
					return nil, connect.NewError(connect.CodeUnavailable, errShut)
				}
				defer s.questions.finish()
				return next(ctx, r)
			}
		},
	))
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
	vault := s.state.GetCurrentVault()
	name := filepath.FromSlash(unescapedPath)
	if vault.Path == "" || !filepath.IsLocal(name) {
		return false
	}
	file, info, err := openVaultFile(vault.Path, name)
	if err != nil {
		return false
	}
	defer closeAndReport(s.state.reportTask, file)
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
	if v := s.state.GetCurrentVault(); v.ID != "" {
		r.Header.Set(requestVaultHeader, string(v.ID))
	}
}

// NewNoteRoute is the route of the notes service.
func NewNoteRoute(state *State, handler *notewindow.Handler) Route {
	path, served := numenv1connect.NewNoteServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewFileRoute is the route of the files service.
func NewFileRoute(state *State, handler *filewindow.Handler) Route {
	path, served := numenv1connect.NewFileServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewMediaRoute is the route of the service that tells where a media file is played from.
func NewMediaRoute(state *State, handler *mediawindow.Handler) Route {
	path, served := numenv1connect.NewMediaServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewSearchRoute is the route of the search service.
func NewSearchRoute(state *State, handler *search.Handler) Route {
	path, served := numenv1connect.NewSearchServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewSettingsRoute is the route of the settings service.
func NewSettingsRoute(state *State, handler *settings.Handler) Route {
	path, served := numenv1connect.NewSettingsServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewTextRoute is the route of the service that reads the text of files.
func NewTextRoute(state *State, handler *texts.Handler) Route {
	path, served := numenv1connect.NewTextServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewCatalogRoute is the route of the service that lists the plugins the application has.
func NewCatalogRoute(state *State, handler *plugincatalog.Handler) Route {
	path, served := numenv1connect.NewCatalogServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}
