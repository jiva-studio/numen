package webui

import (
	"embed"
	"net/http"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/appearance"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

//go:embed all:pages
var pages embed.FS

// policy is what this window may load. A page of a document arrives as a
// picture at a URL of its own, so nothing here draws from anywhere but itself.
var policy = appearance.Policy()

// Pages is the interface itself, built by `make interface` and carried inside
// the binary. A binary built without it says so.
func Pages() (http.Handler, error) { return appearance.Serving(pages) }

// Serving puts the questions in front of the pages, so that a window and a
// browser are answered by one handler.
func (a *API) Serving(files http.Handler) http.Handler {
	route, questions := numenv1connect.NewVaultServiceHandler(a)
	asking, tasks := numenv1connect.NewAgentServiceHandler(a)
	wearing, themes := numenv1connect.NewThemeServiceHandler(a.dressed())
	listing, held := numenv1connect.NewVaultsServiceHandler(vaults{api: a})
	cutting, decks := numenv1connect.NewCardsServiceHandler(a)
	scheduling, presets := numenv1connect.NewPresetsServiceHandler(a)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", policy)
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
			a.Asset(w, r)
		case slices.Contains(appearance.OpenedAt, r.URL.Path):
			appearance.Window(w, r, pages, a.Themes, files)
		default:
			files.ServeHTTP(w, r)
		}
	})
}
