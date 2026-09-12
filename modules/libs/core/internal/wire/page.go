package wire

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/appearance"
)

// OpensAt is the document a window opens at.
const OpensAt = "index.html"

// OpenedAt are the addresses a window is handed for the page itself. A URL
// parses to none of them empty, and a window is asked for under all three.
var OpenedAt = []string{"", "/", "/" + OpensAt}

// GetInterface is the interface as it sits inside a binary. A binary built
// without one says so, in the words of the person who has to fix it.
func GetInterface(pages fs.FS) (fs.FS, error) {
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

// NewInterfaceServer is the built interface as a file server.
func NewInterfaceServer(pages fs.FS) (http.Handler, error) {
	built, err := GetInterface(pages)
	if err != nil {
		return nil, err
	}
	return http.FileServerFS(built), nil
}

// Page writes the document a window opens at, already showing what the settings
// say. The bytes are written here and kept nowhere, so no frame is drawn in the
// default colours or at a size nobody asked for.
//
// A build carrying no interface, and a page with no head, are served as the
// file server has them.
func Page(
	w http.ResponseWriter,
	r *http.Request,
	pages fs.FS,
	themes numenv1connect.ThemeServiceHandler,
	files http.Handler,
) {
	built, err := GetInterface(pages)
	if err != nil {
		files.ServeHTTP(w, r)
		return
	}
	text, err := fs.ReadFile(built, OpensAt)
	if err != nil {
		files.ServeHTTP(w, r)
		return
	}
	if said, is := readAppearanceSettings(r.Context(), themes); is {
		text = appearance.Into(text, said.Styles())
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(text)))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(text)
}

// readAppearanceSettings is what the settings say a window shows, and whether
// they could say anything at all.
//
// It is asked for every request, so a theme chosen, a size chosen, or a file in
// the person's folder edited, shows on the next reload. Settings that cannot
// say what they hold put nothing in the page, and the tokens the build carries
// stand.
func readAppearanceSettings(
	ctx context.Context, themes numenv1connect.ThemeServiceHandler,
) (appearance.Settings, bool) {
	if themes == nil {
		return appearance.Settings{}, false
	}
	worn, err := themes.ListThemes(ctx, connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		return appearance.Settings{}, false
	}
	out := appearance.Settings{
		Mode:           ModeIn(worn.Msg.GetMode()),
		InterfaceScale: worn.Msg.GetInterfaceScale(),
		TextScale:      worn.Msg.GetTextScale(),
	}
	text, err := themes.ReadTheme(ctx, connect.NewRequest(&v1.ReadThemeRequest{Name: worn.Msg.GetApplied()}))
	if err == nil {
		out.Theme = text.Msg.GetCss()
	}
	return out, true
}
