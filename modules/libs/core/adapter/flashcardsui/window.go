package flashcardsui

import (
	"context"
	"io/fs"
	"net/http"
	"strconv"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/appearance"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// openedAt are the addresses the window is handed for the page itself.
var openedAt = []string{"", "/", "/" + opensAt}

// Window is the page, already wearing what the settings say. The bytes are
// written here and kept nowhere, so no frame shows the default colours or a
// size nobody asked for.
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
