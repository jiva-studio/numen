package reviewui

import (
	"bytes"
	"context"
	"io/fs"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// openedAt are the addresses the window is handed for the page itself.
var openedAt = []string{"", "/", "/" + opensAt}

// headEnd is where the style elements go: after everything the build put in the
// head, so what a person chose is the last word on it.
const headEnd = "</head>"

// Window is the page, already wearing what the settings say. The bytes are
// written here and kept nowhere, so no frame shows the default colours or a
// size nobody asked for.
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

// dress is what the window wears: which half of a colour pair the tokens are
// read as, the theme's own file, and the two sizes it is drawn and set at.
//
// It is read for every request, so a theme chosen, a size chosen, or a file in
// the person's folder edited, is worn by the next reload. A build that cannot
// say what it wears dresses the page in nothing, and `tokens.css` stands.
func (a *API) dress(ctx context.Context) string {
	if a.Themes == nil {
		return ""
	}
	worn, err := a.Themes.Themes(ctx, connect.NewRequest(&v1.ThemesRequest{}))
	if err != nil {
		return ""
	}

	// The mode first and the theme second. A theme pinning `color-scheme` is
	// the later of two declarations weighing the same.
	dressed := styled(":root { color-scheme: " + scheme(worn.Msg.GetMode()) + "; }")
	text, err := a.Themes.Theme(ctx, connect.NewRequest(&v1.ThemeRequest{Name: worn.Msg.GetApplied()}))
	if err == nil && text.Msg.GetCss() != "" {
		dressed += styled(text.Msg.GetCss())
	}
	return dressed + sized(worn.Msg.GetInterfaceScale(), worn.Msg.GetTextScale())
}

// sized is the two multipliers as the page carries them: how large the
// interface is drawn, and how large the text a person reads is set.
func sized(drawn, set float64) string {
	var held []string
	if drawn > 0 {
		held = append(held, "--numen-interface-scale: "+number(drawn))
	}
	if set > 0 {
		held = append(held, "--numen-text-scale: "+number(set))
	}
	if len(held) == 0 {
		return ""
	}
	return styled(":root { " + strings.Join(held, "; ") + "; }")
}

func number(size float64) string { return strconv.FormatFloat(size, 'f', -1, 64) }

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
