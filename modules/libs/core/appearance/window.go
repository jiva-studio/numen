package appearance

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// OpensAt is the document a window opens at.
const OpensAt = "index.html"

// OpenedAt are the addresses a window is handed for the page itself. A URL
// parses to none of them empty, and a window is asked for under all three.
var OpenedAt = []string{"", "/", "/" + OpensAt}

// Policy is what a window may load: what its own handler serves, and nothing
// else. No script runs that it did not serve, no handler written in an
// attribute runs at all, no form is submitted anywhere, and nothing reaches off
// the machine.
//
// Inline style is allowed because a page positions what it draws through the
// style attribute. Pictures names what a window may draw a picture from besides
// itself: a card carries its own bytes, and a `data:` URI is no request.
func Policy(pictures ...string) string {
	from := strings.Join(append([]string{"'self'"}, pictures...), " ")
	return "default-src 'self'; img-src " + from + "; style-src 'self' 'unsafe-inline'; " +
		"font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; " +
		"form-action 'none'; frame-ancestors 'none'"
}

// Built is the interface as it sits inside a binary. A binary built without one
// says so, in the words of the person who has to fix it.
func Built(pages fs.FS) (fs.FS, error) {
	missing := fmt.Errorf("no interface in this binary — run: make interface")
	built, err := fs.Sub(pages, "pages/app")
	if err != nil {
		return nil, missing
	}
	if _, err := fs.Stat(built, OpensAt); err != nil {
		return nil, missing
	}
	return built, nil
}

// Serving is the built interface as a file server.
func Serving(pages fs.FS) (http.Handler, error) {
	built, err := Built(pages)
	if err != nil {
		return nil, err
	}
	return http.FileServerFS(built), nil
}

// Window writes the page already showing what the settings say. The bytes are
// written here and kept nowhere, so no frame is drawn in the default colours or
// at a size nobody asked for.
//
// A build carrying no interface, and a page with no head, are served as the
// file server has them.
func Window(
	w http.ResponseWriter,
	r *http.Request,
	pages fs.FS,
	themes numenv1connect.ThemeServiceHandler,
	files http.Handler,
) {
	built, err := Built(pages)
	if err != nil {
		files.ServeHTTP(w, r)
		return
	}
	text, err := fs.ReadFile(built, OpensAt)
	if err != nil {
		files.ServeHTTP(w, r)
		return
	}
	if said, is := chosen(r.Context(), themes); is {
		text = Into(text, Styles(said))
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(text)))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(text)
}

// chosen is what the settings say a window shows, and whether they could say
// anything at all.
//
// It is asked for every request, so a theme chosen, a size chosen, or a file in
// the person's folder edited, shows on the next reload. Settings that cannot
// say what they hold put nothing in the page, and the tokens the build carries
// stand.
func chosen(
	ctx context.Context, themes numenv1connect.ThemeServiceHandler,
) (Chosen, bool) {
	if themes == nil {
		return Chosen{}, false
	}
	worn, err := themes.Themes(ctx, connect.NewRequest(&v1.ThemesRequest{}))
	if err != nil {
		return Chosen{}, false
	}
	out := Chosen{
		Mode:  mode(worn.Msg.GetMode()),
		Drawn: worn.Msg.GetInterfaceScale(),
		Set:   worn.Msg.GetTextScale(),
	}
	text, err := themes.Theme(ctx, connect.NewRequest(&v1.ThemeRequest{Name: worn.Msg.GetApplied()}))
	if err == nil {
		out.Theme = text.Msg.GetCss()
	}
	return out, true
}

// mode is which half of every colour pair the tokens are read as.
func mode(said v1.Mode) Mode {
	switch said {
	case v1.Mode_MODE_LIGHT:
		return Light
	case v1.Mode_MODE_DARK:
		return Dark
	}
	return System
}
