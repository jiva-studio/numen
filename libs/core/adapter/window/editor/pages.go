package editor

import (
	"context"
	"embed"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/internal/csp"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

//go:embed all:pages
var pages embed.FS

// errShut is a question asked of a window that has been shut, which answers no
// more of them.
var errShut = errors.New("this window is shut")

// Pages is the interface itself, built by `make interface` and carried inside
// the binary. A binary built without it says so.
func Pages() (http.Handler, error) { return wire.NewInterfaceServer(pages) }

// PluginServices is where the calls for the services plugins serve are forwarded.
type PluginServices = interface {
	http.Handler
	// Serves reports whether an enabled plugin serves the service the path names.
	Serves(urlPath string) bool
}

// served is one service this build answers: where its calls arrive, and what
// answers them.
type served struct {
	posVal string
	to     http.Handler
}

// mount takes a generated handler and the path it answers under as one.
func mount(pos string, to http.Handler) served { return served{posVal: pos, to: to} }

// NewHandler puts the questions in front of the pages, so that a window and a
// browser are answered by one handler.
//
// named is the services this build answers, by the names the schema gives them.
// A service left out is not mounted, and a call of one is unanswered because
// nothing serves it — not because a handler standing there has nothing behind
// it. Naming none is a build that answers the whole schema.
func (s *Services) NewHandler(files http.Handler, named ...string) http.Handler {
	routes := s.newRoutes(named)
	// The bytes of a vault file are served only where documents are answered about.
	hasAssets := isServed(named, numenv1connect.DocumentServiceName) || isServed(named, numenv1connect.BookServiceName)

	// Where a recording is played from is known once the socket it is served
	// over is open, which is before a page is ever asked for.
	policy := csp.Sources{
		Media:   s.Playing.getPlayOrigins(),
		Connect: []string{"https://dl.numen.md", "https://numen-dl.b-cdn.net"},
	}.Policy()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", policy)
		// A window being taken away answers nothing.
		if s.state.isClosed() {
			http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
			return
		}
		// A service served by an enabled plugin takes precedence in a build that answers the whole schema.
		isAgent := strings.HasPrefix(r.URL.Path, "/numen.v1.AgentService/")
		if !isAgent && len(named) == 0 && s.PluginServices != nil && s.PluginServices.Serves(r.URL.Path) {
			s.PluginServices.ServeHTTP(w, r)
			return
		}
		for _, one := range routes {
			if strings.HasPrefix(r.URL.Path, one.posVal) {
				one.to.ServeHTTP(w, r)
				return
			}
		}
		s.serveBeside(w, r, hasAssets, files)
	})
}

// isServed is whether a build naming these services answers one. Naming none is
// a build that answers the whole schema.
func isServed(named []string, service string) bool {
	return len(named) == 0 || slices.Contains(named, service)
}

// service is one service of the schema, and how its calls are answered.
type service struct {
	name   string
	handle func(counted connect.HandlerOption) (string, http.Handler)
}

// newRoutes mounts each service this build answers, behind the count of the
// questions asked.
func (s *Services) newRoutes(named []string) []served {
	counted := s.newQuestionCounter()
	var routes []served
	for _, one := range s.newServices() {
		if isServed(named, one.name) {
			routes = append(routes, mount(one.handle(counted)))
		}
	}
	// The themes belong to the installation and arrive here from whatever put
	// the window together, so a build put together without a catalogue serves
	// none.
	if s.Themes != nil && isServed(named, numenv1connect.ThemeServiceName) {
		routes = append(routes, mount(numenv1connect.NewThemeServiceHandler(s.Themes, counted)))
	}
	return routes
}

// newServices is every service the window itself answers.
func (s *Services) newServices() []service {
	return []service{
		{numenv1connect.VaultServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewVaultServiceHandler(s.Vault, c)
		}},
		{numenv1connect.WorkspaceServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewWorkspaceServiceHandler(s.Workspace, c)
		}},
		{numenv1connect.FileServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewFileServiceHandler(s.File, c)
		}},
		{numenv1connect.NoteServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewNoteServiceHandler(s.Note, c)
		}},
		{numenv1connect.SearchServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewSearchServiceHandler(s.Search, c)
		}},
		{numenv1connect.VaultsServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewVaultsServiceHandler(s.Vaults, c)
		}},
		{numenv1connect.SettingsServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewSettingsServiceHandler(s.Settings, c)
		}},
		{numenv1connect.WindowServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewWindowServiceHandler(s.state.Window, c)
		}},
		{numenv1connect.RecordingServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewRecordingServiceHandler(s.Recording, c)
		}},
		{numenv1connect.AgentServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewAgentServiceHandler(s.Agent, c)
		}},
		{numenv1connect.CommandServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewCommandServiceHandler(s.Command, c)
		}},
		{numenv1connect.TextVersionServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewTextVersionServiceHandler(s.TextVersion, c)
		}},
		{numenv1connect.PluginServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			if s.Plugin == nil {
				return numenv1connect.NewPluginServiceHandler(numenv1connect.UnimplementedPluginServiceHandler{}, c)
			}
			return numenv1connect.NewPluginServiceHandler(s.Plugin, c)
		}},
	}
}

// updateRoute is the endpoint the window asks for release updates.
const updateRoute = "/api/app-update/latest.json"

// updateEndpoint is where the published release manifest sits.
const updateEndpoint = "https://dl.numen.md/latest/latest.json"

// serveBeside answers what no service does: the bytes of a file, the page the
// interface opens at, and every other file the interface is made of.
func (s *Services) serveBeside(w http.ResponseWriter, r *http.Request, hasAssets bool, files http.Handler) {
	switch {
	case r.URL.Path == updateRoute:
		s.serveUpdate(w, r)
	case strings.HasPrefix(r.URL.Path, wire.PluginAssetsRoute):
		if !s.state.questions.begin() {
			http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
			return
		}
		defer s.state.questions.finish()
		wire.ServePluginAsset(w, r, s.getPluginDir)
	case hasAssets && strings.HasPrefix(r.URL.Path, assetsRoute):
		if !s.state.questions.begin() {
			http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
			return
		}
		defer s.state.questions.finish()
		s.serveAsset(w, r)
	case slices.Contains(wire.OpenedAt, r.URL.Path):
		wire.WritePage(w, r, pages, s.Themes, files)
	default:
		files.ServeHTTP(w, r)
	}
}

// serveUpdate proxies the release manifest to the window.
func (s *Services) serveUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, updateEndpoint, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// newQuestionCounter takes every question a client asks and gives it back when it is
// answered, so the index closes with nothing reading it.
//
// A stream is left out: it lives as long as the page that opened it, and the
// window closes while its pages are still drawn.
func (s *Services) newQuestionCounter() connect.HandlerOption {
	return connect.WithInterceptors(connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, r connect.AnyRequest) (connect.AnyResponse, error) {
				if !s.state.questions.begin() {
					return nil, connect.NewError(connect.CodeUnavailable, errShut)
				}
				defer s.state.questions.finish()
				return next(ctx, r)
			}
		},
	))
}

// assetsRoute is the path prefix where document pages and vault file assets are served.
const assetsRoute = "/assets/"

type localHTTPClient struct {
	handler http.Handler
}

func (c *localHTTPClient) Do(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}

func parseAssetPage(r *http.Request) (string, int32, int32, bool) {
	escaped := r.URL.EscapedPath()
	rest := strings.TrimPrefix(escaped, assetsRoute)
	if rest == "" || rest == escaped {
		return "", 0, 0, false
	}
	idx := strings.Index(rest, "/pages/")
	if idx == -1 {
		return "", 0, 0, false
	}
	pathEscaped := rest[:idx]
	pageStr := rest[idx+len("/pages/"):]
	if slashIdx := strings.Index(pageStr, "/"); slashIdx != -1 {
		pageStr = pageStr[:slashIdx]
	}
	pathUnescaped, err := url.PathUnescape(pathEscaped)
	if err != nil || pathUnescaped == "" {
		return "", 0, 0, false
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 0 {
		return "", 0, 0, false
	}
	wide, _ := strconv.Atoi(r.URL.Query().Get("wide"))
	if wide < 0 {
		wide = 0
	}
	return pathUnescaped, int32(page), int32(wide), true
}

func (s *Services) serveAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path, page, width, isPage := parseAssetPage(r)
	if isPage {
		s.serveDocumentPage(w, r, path, page, width)
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

func (s *Services) serveDocumentPage(w http.ResponseWriter, r *http.Request, path string, page, width int32) {
	if s.PluginServices == nil {
		http.Error(w, "no document plugin available", http.StatusServiceUnavailable)
		return
	}
	client := numenv1connect.NewDocumentServiceClient(&localHTTPClient{handler: s.PluginServices}, "http://local")
	resp, err := client.RenderPage(r.Context(), connect.NewRequest(&v1.RenderPageRequest{
		Path:  path,
		Page:  page,
		Width: width,
	}))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	contentType := resp.Msg.ContentType
	if contentType == "" {
		contentType = "image/png"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.Msg.ImageData)
}

func (s *Services) serveVaultAsset(w http.ResponseWriter, r *http.Request, unescapedPath string) bool {
	vault := s.state.GetCurrentVault()
	if vault.Path == "" {
		return false
	}
	cleanVault := filepath.Clean(vault.Path)
	cleanTarget := filepath.Clean(filepath.Join(vault.Path, unescapedPath))
	if !strings.HasPrefix(cleanTarget, cleanVault) {
		return false
	}
	info, err := os.Stat(cleanTarget)
	if err != nil || info.IsDir() {
		return false
	}
	http.ServeFile(w, r, cleanTarget)
	return true
}
