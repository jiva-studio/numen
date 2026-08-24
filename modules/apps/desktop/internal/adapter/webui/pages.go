package webui

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

//go:embed all:pages
var pages embed.FS

// Pages is the interface itself, built by `make interface` and carried inside
// the binary. A binary built without it says so.
func Pages() (http.Handler, error) {
	built, err := fs.Sub(pages, "pages/app")
	if err != nil {
		return nil, fmt.Errorf("no interface in this binary — run: make interface")
	}
	if _, err := fs.Stat(built, "index.html"); err != nil {
		return nil, fmt.Errorf("no interface in this binary — run: make interface")
	}
	return http.FileServerFS(built), nil
}

// policy is what the window may load: what this handler serves, and nothing
// else. A page of a document arrives as a picture at a URL of its own, so
// nothing here needs `data:` or `blob:`.
//
// Inline style is allowed because a page positions what it draws through the
// style attribute.
const policy = "default-src 'self'; img-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; " +
	"frame-ancestors 'none'"

// Serving puts the questions in front of the pages, so that a window and a
// browser are answered by one handler.
func (a *API) Serving(files http.Handler) http.Handler {
	route, questions := numenv1connect.NewVaultServiceHandler(a)
	asking, tasks := numenv1connect.NewAgentServiceHandler(a)
	dressing, themes := numenv1connect.NewThemeServiceHandler(a.dressed())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", policy)
		switch {
		case strings.HasPrefix(r.URL.Path, route):
			questions.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, asking):
			tasks.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, dressing):
			themes.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.EscapedPath(), assetsRoute):
			a.Asset(w, r)
		default:
			files.ServeHTTP(w, r)
		}
	})
}
