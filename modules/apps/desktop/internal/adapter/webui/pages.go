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
// the binary. A binary built without it says so rather than opening a window
// onto nothing.
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

// Serving puts the questions in front of the pages, so that a window and a
// browser are answered by one handler.
func (a *API) Serving(files http.Handler) http.Handler {
	route, questions := numenv1connect.NewVaultServiceHandler(a)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, route) {
			questions.ServeHTTP(w, r)
			return
		}
		files.ServeHTTP(w, r)
	})
}
