package webui

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

//go:embed all:pages
var pages embed.FS

// opensAt is the document the window opens at.
const opensAt = "index.html"

// Pages is the interface itself, built by `make interface` and carried inside
// the binary. A binary built without it says so.
func Pages() (http.Handler, error) {
	built, err := interfaceIn()
	if err != nil {
		return nil, err
	}
	return http.FileServerFS(built), nil
}

// interfaceIn is the built interface as it sits in the binary.
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
	listing, held := numenv1connect.NewVaultsServiceHandler(vaults{api: a})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", policy)
		switch {
		case strings.HasPrefix(r.URL.Path, listing):
			held.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, route):
			questions.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, asking):
			tasks.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, dressing):
			themes.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.EscapedPath(), assetsRoute):
			a.Asset(w, r)
		case slices.Contains(openedAt, r.URL.Path):
			a.Window(w, r, files)
		default:
			files.ServeHTTP(w, r)
		}
	})
}

// openedAt are the addresses the window is handed for the page itself.
var openedAt = []string{"", "/", "/" + opensAt}

// headEnd is where the two style elements go: after everything the build put in
// the head, the built stylesheet's link last among them.
const headEnd = "</head>"

// Window is the page, already wearing what the settings say. The bytes are
// written here and kept nowhere, so no frame shows the default colours and a
// theme chosen shows on the next reload.
//
// A build carrying no interface, and a page with no head, are served as the
// file server has them.
func (a *API) Window(w http.ResponseWriter, r *http.Request, files http.Handler) {
	built, err := interfaceIn()
	if err != nil {
		files.ServeHTTP(w, r)
		return
	}
	text, err := fs.ReadFile(built, opensAt)
	if err != nil {
		files.ServeHTTP(w, r)
		return
	}

	at := bytes.LastIndex(text, []byte(headEnd))
	if dress := a.dress(r.Context()); dress != "" && at >= 0 {
		text = slices.Concat(text[:at], []byte(dress), text[at:])
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(text)))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(text)
}

// dress is what the window wears, as the two elements the head ends with: which
// half of a colour pair the tokens are read as, and the theme's own file.
//
// It is read for every request, so a theme chosen, or a file in the person's
// folder edited, is worn by the next reload. A build that cannot say what it
// wears dresses the page in nothing, and `tokens.css` stands.
func (a *API) dress(ctx context.Context) string {
	if a.Themes == nil {
		return ""
	}
	worn, err := a.Themes.Themes(ctx, connect.NewRequest(&v1.ThemesRequest{}))
	if err != nil {
		return ""
	}

	// The mode first and the theme second. A theme pinning `color-scheme` is
	// the later of two declarations weighing the same, and light and dark are
	// then that theme's own.
	dressed := styled(":root { color-scheme: " + scheme(worn.Msg.GetMode()) + "; }")
	text, err := a.Themes.Theme(ctx, connect.NewRequest(&v1.ThemeRequest{Name: worn.Msg.GetApplied()}))
	if err != nil || text.Msg.GetCss() == "" {
		return dressed
	}
	return dressed + styled(text.Msg.GetCss())
}

// scheme is which half of every colour pair the tokens are read as.
func scheme(mode v1.Mode) string {
	switch mode {
	case v1.Mode_MODE_LIGHT:
		return "light"
	case v1.Mode_MODE_DARK:
		return "dark"
	}
	return "light dark"
}

// styled is one stylesheet as the page carries it. A `</style>` in a theme's
// file is written as the CSS escape for it, which is the same declaration and
// ends no element.
func styled(css string) string {
	escaped := ending.ReplaceAllStringFunc(css, func(found string) string {
		return `<\` + found[len("<"):]
	})
	return "<style>" + escaped + "</style>\n"
}

var ending = regexp.MustCompile(`(?i)</style`)
