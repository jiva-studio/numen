package editor

import (
	"context"
	"embed"
	"errors"
	"net/http"
	"slices"
	"strings"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/csp"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

//go:embed all:pages
var pages embed.FS

// errShut is a question asked of a window that has been shut, which answers no
// more of them.
var errShut = errors.New("this window is shut")

// Pages is the interface itself, built by `make interface` and carried inside
// the binary. A binary built without it says so.
func Pages() (http.Handler, error) { return wire.Serving(pages) }

// served is one service this build answers: where its calls arrive, and what
// answers them.
type served struct {
	at string
	to http.Handler
}

// mount takes a generated handler and the path it answers under as one.
func mount(at string, to http.Handler) served { return served{at: at, to: to} }

// Serving puts the questions in front of the pages, so that a window and a
// browser are answered by one handler.
//
// named is the services this build answers, by the names the schema gives them.
// A service left out is not mounted, and a call of one is unanswered because
// nothing serves it — not because a handler standing there has nothing behind
// it. Naming none is a build that answers the whole schema.
func (a *API) Serving(files http.Handler, named ...string) http.Handler {
	counted := a.counting()
	serves := func(service string) bool {
		return len(named) == 0 || slices.Contains(named, service)
	}

	var routes []served
	if serves(numenv1connect.VaultServiceName) {
		routes = append(routes, mount(numenv1connect.NewVaultServiceHandler(a, counted)))
	}
	if serves(numenv1connect.WorkspaceServiceName) {
		routes = append(routes, mount(numenv1connect.NewWorkspaceServiceHandler(a, counted)))
	}
	if serves(numenv1connect.FileServiceName) {
		routes = append(routes, mount(numenv1connect.NewFileServiceHandler(a, counted)))
	}
	if serves(numenv1connect.NoteServiceName) {
		routes = append(routes, mount(numenv1connect.NewNoteServiceHandler(a, counted)))
	}
	if serves(numenv1connect.SearchServiceName) {
		routes = append(routes, mount(numenv1connect.NewSearchServiceHandler(a, counted)))
	}
	if serves(numenv1connect.AgentServiceName) {
		routes = append(routes, mount(numenv1connect.NewAgentServiceHandler(a, counted)))
	}
	if serves(numenv1connect.VaultsServiceName) {
		routes = append(routes, mount(numenv1connect.NewVaultsServiceHandler(vaultsService{api: a}, counted)))
	}
	if serves(numenv1connect.CardsServiceName) {
		routes = append(routes, mount(numenv1connect.NewCardsServiceHandler(a, counted)))
	}
	if serves(numenv1connect.PresetsServiceName) {
		routes = append(routes, mount(numenv1connect.NewPresetsServiceHandler(a, counted)))
	}
	if serves(numenv1connect.SettingsServiceName) {
		routes = append(routes, mount(numenv1connect.NewSettingsServiceHandler(a, counted)))
	}
	if serves(numenv1connect.WindowServiceName) {
		routes = append(routes, mount(numenv1connect.NewWindowServiceHandler(a.Window, counted)))
	}
	if serves(numenv1connect.ArtifactServiceName) {
		routes = append(routes, mount(numenv1connect.NewArtifactServiceHandler(a, counted)))
	}
	if serves(numenv1connect.AssetServiceName) {
		routes = append(routes, mount(numenv1connect.NewAssetServiceHandler(a, counted)))
	}
	// The themes belong to the installation and arrive here from whatever put
	// the window together, so a build put together without a catalogue serves
	// none.
	if a.Themes != nil && serves(numenv1connect.ThemeServiceName) {
		routes = append(routes, mount(numenv1connect.NewThemeServiceHandler(a.Themes, counted)))
	}

	// A file's own bytes are what a browser's own elements speak, and they are
	// the file, so they are served where the file is answered about and nowhere
	// else.
	bytes := serves(numenv1connect.AssetServiceName)

	// Where a recording is played from is known once the socket it is served
	// over is open, which is before a page is ever asked for. What a link note
	// points at is played in a frame, from the hosts named here and no other.
	// A player is framed from this run's own socket and never from the host
	// directly: a host is told which address holds its player, and a window
	// drawn from a scheme of its own has none to give.
	policy := csp.Sources{
		Media:  a.Playing.named(),
		Frames: a.Playing.named(),
	}.Policy()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", policy)
		// A window being taken away answers nothing.
		if a.closed() {
			http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
			return
		}
		for _, one := range routes {
			if strings.HasPrefix(r.URL.Path, one.at) {
				one.to.ServeHTTP(w, r)
				return
			}
		}
		switch {
		case bytes && strings.HasPrefix(r.URL.EscapedPath(), assetsRoute):
			if !a.questions.begin() {
				http.Error(w, errShut.Error(), http.StatusServiceUnavailable)
				return
			}
			defer a.questions.done()
			a.Asset(w, r)
		case slices.Contains(wire.OpenedAt, r.URL.Path):
			wire.Page(w, r, pages, a.Themes, files)
		default:
			files.ServeHTTP(w, r)
		}
	})
}

// counting takes every question a client asks and gives it back when it is
// answered, so the index closes with nothing reading it.
//
// A stream is left out: it lives as long as the page that opened it, and the
// window closes while its pages are still drawn.
func (a *API) counting() connect.HandlerOption {
	return connect.WithInterceptors(connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, r connect.AnyRequest) (connect.AnyResponse, error) {
				if !a.questions.begin() {
					return nil, connect.NewError(connect.CodeUnavailable, errShut)
				}
				defer a.questions.done()
				return next(ctx, r)
			}
		},
	))
}
