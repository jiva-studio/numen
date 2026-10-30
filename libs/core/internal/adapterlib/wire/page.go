package wire

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/appearance"
)

// OpensAt is the document a window opens at.
const OpensAt = "index.html"

// OpenedAt lists the addresses under which a window requests the page itself:
// empty, "/" and "/index.html".
var OpenedAt = []string{"", "/", "/" + OpensAt}

// GetInterface returns the built interface, or an error explaining how to build
// it when the binary has none.
func GetInterface(pages fs.FS) (fs.FS, error) {
	missing := fmt.Errorf("no interface in this binary — run: make interface")
	constructed, err := fs.Sub(pages, "pages/app")
	if err != nil {
		return nil, missing
	}
	if _, err := fs.Stat(constructed, OpensAt); err != nil {
		return nil, missing
	}
	return constructed, nil
}

// NewInterfaceServer is the built interface as a file server.
func NewInterfaceServer(pages fs.FS) (http.Handler, error) {
	constructed, err := GetInterface(pages)
	if err != nil {
		return nil, err
	}
	return http.FileServerFS(constructed), nil
}

// WritePage writes the document a window opens at, with the current settings
// already applied. The bytes are generated per request and not cached, so no
// frame renders with default colors or an unrequested size.
//
// A build without an interface, and a page without a head, are served as the
// file server provides them.
func WritePage(
	w http.ResponseWriter,
	r *http.Request,
	pages fs.FS,
	themes numenv1connect.ThemeServiceHandler,
	files http.Handler,
) {
	constructed, err := GetInterface(pages)
	if err != nil {
		files.ServeHTTP(w, r)
		return
	}
	text, err := fs.ReadFile(constructed, OpensAt)
	if err != nil {
		files.ServeHTTP(w, r)
		return
	}
	if response, is := readAppearanceSettings(r.Context(), themes); is {
		text = appearance.AddStyles(text, response.Styles())
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(text)))
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(text))
}

// readAppearanceSettings returns the appearance the settings specify for a
// window, and whether the settings could be read.
//
// It runs on every request, so a changed theme, a changed size, or an edited
// file in the user's folder takes effect on the next reload. When the settings
// cannot be read, nothing is added to the page and the tokens built into the
// bundle apply.
func readAppearanceSettings(
	ctx context.Context, themes numenv1connect.ThemeServiceHandler,
) (appearance.Settings, bool) {
	if themes == nil {
		return appearance.Settings{}, false
	}
	aged, err := themes.ListThemes(ctx, connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		return appearance.Settings{}, false
	}
	out := appearance.Settings{
		Mode:           ConvertModeFromWire(aged.Msg.GetMode()),
		InterfaceScale: aged.Msg.GetInterfaceScale(),
		TextScale:      aged.Msg.GetTextScale(),
	}
	text, err := themes.ReadTheme(ctx, connect.NewRequest(&v1.ReadThemeRequest{Name: aged.Msg.GetApplied()}))
	if err == nil {
		out.Theme = text.Msg.GetCss()
	}
	return out, true
}
