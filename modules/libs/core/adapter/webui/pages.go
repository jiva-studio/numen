package webui

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/appearance"
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
	wearing, themes := numenv1connect.NewThemeServiceHandler(a.dressed())
	listing, held := numenv1connect.NewVaultsServiceHandler(vaults{api: a})
	cutting, decks := numenv1connect.NewCardsServiceHandler(a)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", policy)
		switch {
		case strings.HasPrefix(r.URL.Path, listing):
			held.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, cutting):
			decks.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, route):
			questions.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, asking):
			tasks.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, wearing):
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

	if said, is := a.chosen(r.Context()); is {
		text = appearance.Into(text, appearance.Styles(said))
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(text)))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(text)
}

// chosen is what the settings say this window shows, and whether they could say
// anything at all.
//
// It is asked for every request, so a theme chosen, a size chosen, or a file in
// the person's folder edited, shows on the next reload. Settings that cannot
// say what they hold put nothing in the page, and the tokens the build carries
// stand.
func (a *API) chosen(ctx context.Context) (appearance.Chosen, bool) {
	if a.Themes == nil {
		return appearance.Chosen{}, false
	}
	worn, err := a.Themes.Themes(ctx, connect.NewRequest(&v1.ThemesRequest{}))
	if err != nil {
		return appearance.Chosen{}, false
	}
	out := appearance.Chosen{
		Mode:  mode(worn.Msg.GetMode()),
		Drawn: worn.Msg.GetInterfaceScale(),
		Set:   worn.Msg.GetTextScale(),
	}
	text, err := a.Themes.Theme(ctx, connect.NewRequest(&v1.ThemeRequest{Name: worn.Msg.GetApplied()}))
	if err == nil {
		out.Theme = text.Msg.GetCss()
	}
	return out, true
}

// mode is which half of every colour pair the tokens are read as.
func mode(said v1.Mode) appearance.Mode {
	switch said {
	case v1.Mode_MODE_LIGHT:
		return appearance.Light
	case v1.Mode_MODE_DARK:
		return appearance.Dark
	}
	return appearance.System
}
