package flashcardsui

import (
	"embed"
	"net/http"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/appearance"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

//go:embed all:pages
var pages embed.FS

// policy is what this window may load.
//
// A card is HTML, and a deck may have come from another person, so the page
// draws it through the allowlist the library keeps and this line stands behind
// that. A picture written into a card is a `data:` URI, which is the card's own
// bytes and no request at all.
var policy = appearance.Policy(appearance.Sources{Images: []string{"data:"}})

// Pages is the interface itself, built by `make interface` and carried inside
// the binary. A binary built without it says so.
func Pages() (http.Handler, error) { return wire.Serving(pages) }

// Serving is the whole of what this window answers: the flashcards service, the
// agent a card is asked about through, the window itself, the themes it is
// dressed from, and the files the page is made of.
func (a *API) Serving(files http.Handler) http.Handler {
	route, questions := numenv1connect.NewFlashcardsServiceHandler(a)
	asking, agent := numenv1connect.NewAgentServiceHandler(a)
	drawn, itself := numenv1connect.NewWindowServiceHandler(a.Window)
	var dressing string
	var themes http.Handler
	if a.Themes != nil {
		dressing, themes = numenv1connect.NewThemeServiceHandler(a.Themes)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", policy)
		switch {
		case strings.HasPrefix(r.URL.Path, route):
			questions.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, asking):
			agent.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, drawn):
			itself.ServeHTTP(w, r)
		case themes != nil && strings.HasPrefix(r.URL.Path, dressing):
			themes.ServeHTTP(w, r)
		case slices.Contains(wire.OpenedAt, r.URL.Path):
			wire.Page(w, r, pages, a.Themes, files)
		default:
			files.ServeHTTP(w, r)
		}
	})
}
