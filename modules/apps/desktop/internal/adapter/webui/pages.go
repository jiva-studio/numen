package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

//go:embed all:pages
var pages embed.FS

// Pages is the interface itself, built by `make interface` and carried inside
// the binary.
func Pages() http.Handler {
	built, err := fs.Sub(pages, "pages")
	if err != nil {
		panic(err)
	}
	return http.FileServerFS(built)
}

// Serving puts the questions in front of the pages, so that a window and a
// browser are answered by one handler.
func (a *API) Serving(files http.Handler) http.Handler {
	route, questions := numenv1connect.NewVaultHandler(a)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, route) {
			questions.ServeHTTP(w, r)
			return
		}
		files.ServeHTTP(w, r)
	})
}
