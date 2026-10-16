package editor

import (
	"context"
	"embed"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/modules/libs/core/internal/csp"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

//go:embed all:pages
var pages embed.FS

// errShut is a question asked of a window that has been shut, which answers no
// more of them.
var errShut = errors.New("this window is shut")

// Pages is the interface itself, built by `make interface` and carried inside
// the binary. A binary built without it says so.
func Pages() (http.Handler, error) { return wire.NewInterfaceServer(pages) }

// served is one service this build answers: where its calls arrive, and what
// answers them.
type served struct {
	at string
	to http.Handler
}

// mount takes a generated handler and the path it answers under as one.
func mount(at string, to http.Handler) served { return served{at: at, to: to} }

// NewHandler puts the questions in front of the pages, so that a window and a
// browser are answered by one handler.
//
// named is the services this build answers, by the names the schema gives them.
// A service left out is not mounted, and a call of one is unanswered because
// nothing serves it — not because a handler standing there has nothing behind
// it. Naming none is a build that answers the whole schema.
func (s *Services) NewHandler(files http.Handler, named ...string) http.Handler {
	routes := s.newRoutes(named)

	// A page of a document, the markup of a book and a picture the book carries
	// are what a browser's own elements speak, and each is the thing itself, so
	// they are served where that thing is answered about and nowhere else.
	hasAssets := isServed(named, numenv1connect.DocumentServiceName) || isServed(named, numenv1connect.BookServiceName)

	// Where a recording is played from is known once the socket it is served
	// over is open, which is before a page is ever asked for. What a link note
	// points at is played in a frame, from the hosts named here and no other.
	// A player is framed from this run's own socket and never from the host
	// directly: a host is told which address holds its player, and a window
	// drawn from a scheme of its own has none to give.
	policy := csp.Sources{
		Media:   s.Playing.getPlayOrigins(),
		Frames:  s.Playing.getPlayOrigins(),
		Connect: []string{"https://dl.numen.md", "https://numen-dl.b-cdn.net"},
	}.Policy()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", policy)
		// A window being taken away answers nothing.
		if s.state.isClosed() {
			http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
			return
		}
		for _, one := range routes {
			if strings.HasPrefix(r.URL.Path, one.at) {
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
		{numenv1connect.AgentServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewAgentServiceHandler(s.Agent, c)
		}},
		{numenv1connect.VaultsServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewVaultsServiceHandler(s.Vaults, c)
		}},
		{numenv1connect.CardsServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewCardsServiceHandler(s.Cards, c)
		}},
		{numenv1connect.PresetsServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewPresetsServiceHandler(s.Presets, c)
		}},
		{numenv1connect.SettingsServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewSettingsServiceHandler(s.Settings, c)
		}},
		{numenv1connect.WindowServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewWindowServiceHandler(s.state.Window, c)
		}},
		{numenv1connect.ArtifactServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewArtifactServiceHandler(s.Artifact, c)
		}},
		{numenv1connect.TranscriptServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewTranscriptServiceHandler(s.Transcript, c)
		}},
		{numenv1connect.ArticleServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewArticleServiceHandler(s.Article, c)
		}},
		{numenv1connect.OcrServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewOcrServiceHandler(s.Ocr, c)
		}},
		{numenv1connect.DocumentServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewDocumentServiceHandler(s.Document, c)
		}},
		{numenv1connect.RecordingServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewRecordingServiceHandler(s.Recording, c)
		}},
		{numenv1connect.BookServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			return numenv1connect.NewBookServiceHandler(s.Book, c)
		}},
		{numenv1connect.PluginServiceName, func(c connect.HandlerOption) (string, http.Handler) {
			if s.Plugin == nil {
				return numenv1connect.NewPluginServiceHandler(numenv1connect.UnimplementedPluginServiceHandler{}, c)
			}
			return numenv1connect.NewPluginServiceHandler(s.Plugin, c)
		}},
	}
}

// pluginAssetsRoute is the path prefix where plugin bundles and assets are served.
const pluginAssetsRoute = "/api/plugins/"

// serveBeside answers what no service does: the bytes of a file, the page the
// interface opens at, and every other file the interface is made of.
func (s *Services) serveBeside(w http.ResponseWriter, r *http.Request, hasAssets bool, files http.Handler) {
	switch {
	case strings.HasPrefix(r.URL.Path, pluginAssetsRoute):
		if !s.state.questions.begin() {
			http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
			return
		}
		defer s.state.questions.finish()
		s.servePluginAsset(w, r)
	case hasAssets && strings.HasPrefix(r.URL.EscapedPath(), assetsRoute):
		if !s.state.questions.begin() {
			http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
			return
		}
		defer s.state.questions.finish()
		s.Asset.Asset(w, r)
	case slices.Contains(wire.OpenedAt, r.URL.Path):
		wire.WritePage(w, r, pages, s.Themes, files)
	default:
		files.ServeHTTP(w, r)
	}
}

func resolvePluginAssetTarget(pluginDir, filePath string) (string, error) {
	cleanDir, err := filepath.Abs(filepath.Clean(pluginDir))
	if err != nil {
		return "", err
	}
	cleanTarget, err := filepath.Abs(filepath.Join(cleanDir, filepath.Clean(filePath)))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(cleanDir, cleanTarget)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", os.ErrPermission
	}
	return cleanTarget, nil
}

func setPluginAssetContentType(w http.ResponseWriter, path string) {
	switch filepath.Ext(path) {
	case ".js", ".mjs":
		w.Header().Set("Content-Type", "application/javascript")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".json":
		w.Header().Set("Content-Type", "application/json")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".wasm":
		w.Header().Set("Content-Type", "application/wasm")
	}
}

// servePluginAsset writes the bytes of a plugin's frontend bundle or stylesheet.
func (s *Services) servePluginAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.getPluginDir == nil {
		http.Error(w, "plugin manager not configured", http.StatusNotFound)
		return
	}
	trimmed := strings.TrimPrefix(r.URL.Path, pluginAssetsRoute)
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "invalid plugin asset path", http.StatusBadRequest)
		return
	}

	pluginDir, ok := s.getPluginDir(parts[0])
	if !ok {
		http.Error(w, "plugin not found", http.StatusNotFound)
		return
	}

	cleanTarget, err := resolvePluginAssetTarget(pluginDir, parts[1])
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			http.Error(w, "forbidden: path escapes plugin directory", http.StatusForbidden)
		} else {
			http.Error(w, "cannot resolve asset path", http.StatusBadRequest)
		}
		return
	}

	info, err := os.Stat(cleanTarget)
	if err != nil || info.IsDir() {
		http.Error(w, "plugin asset not found", http.StatusNotFound)
		return
	}

	setPluginAssetContentType(w, cleanTarget)
	http.ServeFile(w, r, cleanTarget)
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
