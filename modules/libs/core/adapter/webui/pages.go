package webui

import (
	"context"
	"embed"
	"errors"
	"net/http"
	"slices"
	"strings"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/appearance"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

//go:embed all:pages
var pages embed.FS

// policy is what this window may load. A page of a document arrives as a
// picture at a URL of its own, so nothing here draws from anywhere but itself.

// errGone is a question asked of a window whose door is shut.
var errGone = errors.New("this window is going")

// Pages is the interface itself, built by `make interface` and carried inside
// the binary. A binary built without it says so.
func Pages() (http.Handler, error) { return appearance.Serving(pages) }

// Serving puts the questions in front of the pages, so that a window and a
// browser are answered by one handler.
func (a *API) Serving(files http.Handler) http.Handler {
	counted := a.counting()
	route, questions := numenv1connect.NewVaultServiceHandler(a, counted)
	asking, tasks := numenv1connect.NewAgentServiceHandler(a, counted)
	wearing, themes := numenv1connect.NewThemeServiceHandler(a.dressed(), counted)
	listing, held := numenv1connect.NewVaultsServiceHandler(vaults{api: a}, counted)
	cutting, decks := numenv1connect.NewCardsServiceHandler(a, counted)
	scheduling, presets := numenv1connect.NewPresetsServiceHandler(a, counted)
	// Where a recording is played from is known once the socket it is served
	// over is open, which is before a page is ever asked for.
	policy := appearance.Policy(appearance.Sources{Media: a.Playing.named()})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", policy)
		// A window being taken away answers nothing.
		if a.closed() {
			http.Error(w, errGone.Error(), http.StatusServiceUnavailable)
			return
		}
		switch {
		case strings.HasPrefix(r.URL.Path, listing):
			held.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, cutting):
			decks.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, scheduling):
			presets.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, route):
			questions.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, asking):
			tasks.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, wearing):
			themes.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.EscapedPath(), assetsRoute):
			if !a.questions.begin() {
				http.Error(w, errGone.Error(), http.StatusServiceUnavailable)
				return
			}
			defer a.questions.done()
			a.Asset(w, r)
		case slices.Contains(appearance.OpenedAt, r.URL.Path):
			appearance.Window(w, r, pages, a.Themes, files)
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
					return nil, connect.NewError(connect.CodeUnavailable, errGone)
				}
				defer a.questions.done()
				return next(ctx, r)
			}
		},
	))
}
