package editor

import (
	"embed"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/localendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"

	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

//go:embed all:pages
var pages embed.FS

// errShut is returned for a request to a window that has been shut; the window
// answers no further requests.
var errShut = wire.ErrShut

// CreatePagesHandler is the interface itself, built by `make interface` and embedded in the
// binary. A binary built without it reports an error.
func CreatePagesHandler() (http.Handler, error) { return wire.NewInterfaceServer(pages) }

// PluginServices is where the calls for the services plugins serve are forwarded.
type PluginServices interface {
	http.Handler
	// CanServe reports whether an enabled plugin serves the service the path names.
	CanServe(urlPath string) bool
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
}

// NewRoute is the route of a service of the schema. Handler is what the schema
// generated for the service.
func NewRoute(path string, handler http.Handler) Route {
	return Route{path: path, handler: handler}
}

// pageSet is what answers the requests that reach a window: the routes of its
// services and the services beside them.
type pageSet struct {
	window       *PageAccess
	routes       []Route
	themes       numenv1connect.ThemeServiceHandler
	plugins      PluginServices
	getPluginDir func(id string) (string, bool)
	assets       *file.OpenVaultFile
}

// NewHandler puts the services in front of the pages, so a window and a browser
// are served by one handler.
// named lists the services this build serves, by the names the schema gives
// them. A service left out is not mounted, so a call to it is unanswered.
// Naming none builds the whole schema.
func (o *Installation) NewHandler(files http.Handler, named ...string) http.Handler {
	return o.pages.newHandler(files, named...)
}

// NewLocalHandler is NewHandler for a handler behind a socket that other processes on the machine can
// reach. It answers only a request that carries the returned token as a bearer credential, and the
// token is new on every call.
func (o *Installation) NewLocalHandler(files http.Handler, named ...string) (handler http.Handler, token string, err error) {
	token, err = localendpoint.NewToken()
	if err != nil {
		return nil, "", err
	}
	return localendpoint.RequireBearerToken(token, o.pages.newHandler(files, named...)), token, nil
}

func (s pageSet) newHandler(files http.Handler, named ...string) http.Handler {
	routes := s.newRoutes(named)
	// Only a build that answers the whole schema serves the bytes of vault files and the plugin route.
	isWhole := len(named) == 0

	// The origin recordings play from is known once the socket it is served over is
	// open, which precedes any page request.
	played := s.window.Media.playing.Load().getPlayOrigins()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", s.getPolicy(played))
		// A window being taken away answers nothing.
		if s.window.Gate.IsClosed() {
			http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
			return
		}
		r = r.WithContext(plugindomain.WithCaller(r.Context(), getWindowCaller(r.Header)))
		// A service served by an enabled plugin takes precedence in a build that answers the whole schema.
		if isWhole && s.plugins != nil && s.plugins.CanServe(r.URL.Path) {
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
	return routes
}

func (s pageSet) serveAdjacent(w http.ResponseWriter, r *http.Request, isWhole bool, files http.Handler) {
	switch {
	case strings.HasPrefix(r.URL.Path, wire.PluginRoute):
		if !s.window.Gate.BeginQuestion() {
			http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
			return
		}
		defer s.window.Gate.FinishQuestion()
		s.servePluginRoute(w, r, isWhole)
	case isWhole && strings.HasPrefix(r.URL.Path, assetsRoute):
		if !s.window.Gate.BeginQuestion() {
			http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
			return
		}
		defer s.window.Gate.FinishQuestion()
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
	vault := s.window.session.GetCurrentVault()
	if vault.Path == "" {
		return false
	}
	opened, err := s.assets.ExecuteLatest(r.Context(), vault, unescapedPath)
	if err != nil {
		return false
	}
	defer closeAndReport(s.window.Window.AddTask, opened.Body)
	setAssetHeaders(w.Header(), opened.MediaType)
	http.ServeContent(w, r, path.Base(opened.Path), opened.ModTime, opened.Body)
	return true
}

func (s pageSet) servePluginRoute(w http.ResponseWriter, r *http.Request, isWhole bool) {
	if wire.ServePluginFile(w, r, s.getPluginDir) {
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
	if v := s.window.session.GetCurrentVault(); v.ID != "" {
		r.Header.Set(requestVaultHeader, string(v.ID))
	}
}

// CallerHeader names the part of the window that makes a request: a tag known to
// getWindowCallerName, or "plugin:" and the id of the plugin whose view asks.
const CallerHeader = "Numen-Caller"

func getWindowCallerName(tag string) (string, bool) {
	switch tag {
	case "chat":
		return "Chat", true
	case "commands":
		return "Commands", true
	case "editor":
		return "Editor", true
	case "graph":
		return "Graph", true
	case "plugins":
		return "Plugins", true
	case "search":
		return "Search", true
	case "settings":
		return "Settings", true
	case "tasks":
		return "Tasks", true
	default:
		return "", false
	}
}

func getWindowCaller(header http.Header) plugindomain.Caller {
	tag := header.Get(CallerHeader)
	if pluginID, isPlugin := strings.CutPrefix(tag, "plugin:"); isPlugin && pluginID != "" {
		return plugindomain.Caller{PluginID: pluginID}
	}
	if who, ok := getWindowCallerName(tag); ok {
		return plugindomain.Caller{Who: who}
	}
	return plugindomain.Caller{Who: plugindomain.WhoWindow}
}
