package reviewui

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

//go:embed all:pages
var pages embed.FS

// opensAt is the document the window opens at.
const opensAt = "index.html"

// policy is what this window may load: what this handler serves, and nothing
// else.
//
// A card is HTML, and a deck may have come from another person, so the page
// draws it through the allowlist the library keeps and this line stands behind
// that: no script runs, no form is submitted anywhere, and nothing a card
// carries reaches off the machine. A picture written into a card is a `data:`
// URI, which is the card's own bytes and no request at all.
//
// Inline style is allowed because the page positions what it draws through the
// style attribute.
const policy = "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; " +
	"font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; " +
	"form-action 'none'; frame-ancestors 'none'"

// Pages is the interface itself, built by `make interface` and carried inside
// the binary. A binary built without it says so.
func Pages() (http.Handler, error) {
	built, err := interfaceIn()
	if err != nil {
		return nil, err
	}
	return http.FileServerFS(built), nil
}

func interfaceIn() (fs.FS, error) {
	missing := fmt.Errorf("no interface in this binary — run: make interface")
	built, err := fs.Sub(pages, "pages/app")
	if err != nil {
		return nil, missing
	}
	if _, err := fs.Stat(built, opensAt); err != nil {
		return nil, missing
	}
	return built, nil
}

// Serving is the whole of what this window answers: the review service, the
// themes it is dressed from, and the files the page is made of.
func (a *API) Serving(files http.Handler) http.Handler {
	route, questions := numenv1connect.NewReviewServiceHandler(a)
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
		case themes != nil && strings.HasPrefix(r.URL.Path, dressing):
			themes.ServeHTTP(w, r)
		case slices.Contains(openedAt, r.URL.Path):
			a.Window(w, r, files)
		default:
			files.ServeHTTP(w, r)
		}
	})
}
