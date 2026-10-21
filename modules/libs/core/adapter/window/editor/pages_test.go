package editor

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/modules/libs/core/internal/appearance"
	"github.com/jiva-studio/numen/modules/libs/core/internal/csp"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// mine is a theme of the person's, carrying a colour nothing else has and
// pinning the dark half.
const mine = ":root{color-scheme:dark;--numen-surface:#010203}"

// newPageHandler is the whole handler as the application hands it over: the questions,
// the themes, and the built page behind them.
func newPageHandler(t *testing.T, cfg container.Config) (http.Handler, numenv1connect.ThemeServiceHandler) {
	t.Helper()

	files, err := Pages()
	if err != nil {
		t.Skipf("no interface in this binary: %v", err)
	}
	themes, err := cfg.Themes(nil)
	if err != nil {
		t.Fatal(err)
	}
	return newThemedServices(themes).NewHandler(files), themes
}

// writeTheme writes a theme in the themes directory.
func writeTheme(t *testing.T, cfg container.Config, name, body string) {
	t.Helper()
	folder := filepath.Join(filepath.Dir(cfg.RegistryPath), "themes")
	if err := os.WriteFile(filepath.Join(folder, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func choose(t *testing.T, themes numenv1connect.ThemeServiceHandler, name string, mode v1.Mode) {
	t.Helper()
	writeAppearance(t, themes, &v1.WriteAppearanceRequest{Name: name, Mode: mode})
}

// writeAppearance is one choice as a client makes it, whatever of it the client names.
func writeAppearance(t *testing.T, themes numenv1connect.ThemeServiceHandler, asked *v1.WriteAppearanceRequest) {
	t.Helper()
	out, err := themes.WriteAppearance(t.Context(), connect.NewRequest(asked))
	if err != nil {
		t.Fatal(err)
	}
	if reason := out.Msg.GetError(); reason != "" {
		t.Fatalf("failed: %s", reason)
	}
}

// size is a size a choice names.
func size(said float64) *float64 { return &said }

// getResponse is what comes back at exactly this path. The page is handed over as
// the empty path as well, which no URL parses to.
func getResponse(handler http.Handler, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.URL.Path = path
	out := httptest.NewRecorder()
	handler.ServeHTTP(out, r)
	return out
}

// The policy the window is held to. What each directive says is `appearance`'s
// to say and is asserted there; what this window widens is asserted here.
//
// A build that opened no socket plays from nowhere, so it widens nothing. A
// theme is an ordinary stylesheet a person may have downloaded, and the policy
// is the whole of what stops one reaching the network: a remote `url()` is
// refused by `img-src` and `font-src`, and the element the theme is spliced
// into is permitted by `style-src`.
func TestTheWindowIsHeldToOnePolicy(t *testing.T) {
	held := csp.Sources{
		Connect: []string{"https://dl.numen.md", "https://numen-dl.b-cdn.net"},
	}.Policy()

	handler := newBareServices().NewHandler(http.NotFoundHandler())
	for _, path := range []string{"", "/", "/index.html", "/built/index.css", assetURL("a.pdf")} {
		if said := getResponse(handler, path).Header().Get("Content-Security-Policy"); said != held {
			t.Errorf("%q is held to %q", path, said)
		}
	}
}

// Every host that may be framed is named here, so one arriving in the list
// arrives in a diff somebody reads. A frame runs that host's own scripts and
// reaches that host's own machines, and nothing else about the policy moves.
//
// The window frames this run's own socket and no host at all. The page that
// socket serves is what frames the host, and it is held to a policy of its own.
func TestTheHostsAWindowMayFrame(t *testing.T) {
	if got := embedHosts; !slices.Equal(got, []string{"https://www.youtube-nocookie.com"}) {
		t.Errorf("the window may frame %q", got)
	}

	said := getResponse(newBareServices().NewHandler(http.NotFoundHandler()), "/").
		Header().Get("Content-Security-Policy")
	if !strings.Contains(said, "frame-src 'self';") {
		t.Errorf("the policy reads %q", said)
	}
	if strings.Contains(said, "youtube") {
		t.Errorf("the window names a host of its own to frame: %q", said)
	}

	// A frame is what widens, and a script is not: what runs in the window is
	// what the window served, which is `default-src`.
	if strings.Contains(said, "script-src") {
		t.Errorf("the policy names where a script comes from: %q", said)
	}
}

// TestEveryServiceTheVaultIsAskedAboutIsMounted. Four services answer about the
// vault a window is showing, and every one of them has to be behind the one
// handler the application hands over. A route nothing is mounted at falls
// through to the pages and is answered not found; a route a service holds
// answers a GET as a method that call does not take.
func TestEveryServiceTheVaultIsAskedAboutIsMounted(t *testing.T) {
	handler := newBareServices().NewHandler(http.NotFoundHandler())
	for _, route := range []string{
		numenv1connect.VaultServiceGetVaultStateProcedure,
		numenv1connect.FileServiceListFilesProcedure,
		numenv1connect.NoteServiceReadNoteProcedure,
		numenv1connect.SearchServiceSearchNamesProcedure,
	} {
		if code := getResponse(handler, route).Code; code != http.StatusMethodNotAllowed {
			t.Errorf("%s answered %d, want %d", route, code, http.StatusMethodNotAllowed)
		}
	}
}

// TestPluginServiceIsMounted asserts that PluginServiceHandler is mounted behind
// the handler the application hands over.
func TestPluginServiceIsMounted(t *testing.T) {
	handler := newBareServices().NewHandler(http.NotFoundHandler())
	route := numenv1connect.PluginServiceGetPluginsProcedure
	if code := getResponse(handler, route).Code; code != http.StatusMethodNotAllowed {
		t.Errorf("%s answered %d, want %d", route, code, http.StatusMethodNotAllowed)
	}
}

// A search, a note and a link are read straight from the index, and the index
// closes behind the door. Every question fails at it, whatever it would
// have reached into.
func TestAWindowBeingTakenAwayAnswersNothing(t *testing.T) {
	services := newBareServices()
	handler := services.NewHandler(http.NotFoundHandler())
	questions, _ := numenv1connect.NewVaultServiceHandler(services.Vault)
	asked := []string{"", "/", "/index.html", assetURL("a.pdf"), questions + "Find"}

	for _, path := range asked {
		if code := getResponse(handler, path).Code; code == http.StatusServiceUnavailable {
			t.Errorf("%q failed at %d with the window still open", path, code)
		}
	}

	services.state.Close()

	for _, path := range asked {
		if code := getResponse(handler, path).Code; code != http.StatusServiceUnavailable {
			t.Errorf("%q was answered %d by a window being taken away", path, code)
		}
	}
}

// The page arrives wearing the theme, at every address it is asked for under.
func TestThePageOpensWearingTheTheme(t *testing.T) {
	cfg := newConfig(t)
	handler, themes := newPageHandler(t, cfg)
	writeTheme(t, cfg, "sea.css", mine)
	choose(t, themes, "mine:sea", v1.Mode_MODE_DARK)

	for _, path := range []string{"", "/", "/index.html"} {
		t.Run(strconv.Quote(path), func(t *testing.T) {
			out := getResponse(handler, path)
			if out.Code != http.StatusOK {
				t.Fatalf("answered %d", out.Code)
			}
			if said := out.Header().Get("Content-Type"); !strings.HasPrefix(said, "text/html") {
				t.Errorf("came back as %q", said)
			}
			if !strings.Contains(out.Body.String(), mine) {
				t.Error("the page is not wearing the theme")
			}
			if said := out.Header().Get("Content-Length"); said != strconv.Itoa(out.Body.Len()) {
				t.Errorf("%s bytes were announced and %d written", said, out.Body.Len())
			}
		})
	}
}

// The mode's element, the theme's and the two sizes are the last three things
// in the head, in that order.
//
// A theme and the built stylesheet both declare `color-scheme` at the root and
// weigh the same, so the later of them holds. The built stylesheet's link is
// the last element the build puts in the head, and the theme goes after the
// mode so that a theme pinning the scheme is the one that holds. The sizes go
// after the theme: they are what a person set this window to, inside the bounds
// each goes to, and the window is drawn at what they say.
func TestTheStyleElementsAreTheLastThingInTheHead(t *testing.T) {
	cfg := newConfig(t)
	handler, themes := newPageHandler(t, cfg)
	writeTheme(t, cfg, "sea.css", mine)
	writeAppearance(t, themes, &v1.WriteAppearanceRequest{
		Name:           "mine:sea",
		Mode:           v1.Mode_MODE_DARK,
		InterfaceScale: size(1.25),
		TextScale:      size(1.5),
	})

	head, _, found := strings.Cut(getResponse(handler, "/").Body.String(), appearance.HeadEnd)
	if !found {
		t.Fatal("the page has no head")
	}

	const drawn = `<style data-appearance="sizes">` +
		":root { --numen-interface-scale: 1.25; --numen-text-scale: 1.5; }</style>"
	link := strings.LastIndex(head, "<link")
	mode := strings.Index(head,
		`<style data-appearance="mode">`+":root { color-scheme: dark; }</style>")
	worn := strings.Index(head, mine)
	sizes := strings.Index(head, drawn)
	if link < 0 || mode < 0 || worn < 0 || sizes < 0 {
		t.Fatalf("the link is at %d, the mode at %d, the theme at %d, the sizes at %d",
			link, mode, worn, sizes)
	}
	if link > mode || mode > worn || worn > sizes {
		t.Errorf("the link is at %d, the mode at %d, the theme at %d, the sizes at %d",
			link, mode, worn, sizes)
	}
	if after := strings.TrimSpace(head[sizes+len(drawn):]); after != "" {
		t.Errorf("the head ends with %q", after)
	}
}

// A window drawn at 1.5 goes on being drawn at 1.5.
func TestAFileNamingTheZoomOpensTheWindowDrawnAtIt(t *testing.T) {
	cfg := newConfig(t)
	file := filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json")
	if err := os.WriteFile(file, []byte(`{"appearance":{"zoom":1.5}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	handler, _ := newPageHandler(t, cfg)

	if !strings.Contains(getResponse(handler, "/").Body.String(), "--numen-interface-scale: 1.5") {
		t.Error("the page is not drawn at what the file says")
	}
}

// The built stylesheet declares `color-scheme` at zero weight alone, so the
// element the page carries stands unopposed whatever a theme says.
func TestTheBuiltStylesheetDoesNotPinTheColourScheme(t *testing.T) {
	built, err := wire.GetInterface(pages)
	if err != nil {
		t.Skipf("no interface in this binary: %v", err)
	}

	sheets, err := fs.Glob(built, "built/*.css")
	if err != nil || len(sheets) == 0 {
		t.Fatalf("the build has no stylesheet: %v", err)
	}
	var declaresScheme bool
	for _, name := range sheets {
		text, err := fs.ReadFile(built, name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(text), ":root{color-scheme") {
			t.Errorf("%s declares the colour scheme at the root", name)
		}
		if regexp.MustCompile(`:where\(:root\)\s*\{[^}]*color-scheme`).Match(text) {
			declaresScheme = true
		}
	}
	if !declaresScheme {
		t.Error("the built stylesheets declare no colour scheme at all")
	}
}

// The head is read for each request: a theme chosen is worn by the next reload.
func TestTheHeadIsReadForEachRequest(t *testing.T) {
	cfg := newConfig(t)
	handler, themes := newPageHandler(t, cfg)
	writeTheme(t, cfg, "sea.css", mine)

	if strings.Contains(getResponse(handler, "/").Body.String(), mine) {
		t.Fatal("the page is wearing a theme nobody chose")
	}
	choose(t, themes, "mine:sea", v1.Mode_MODE_DARK)
	if !strings.Contains(getResponse(handler, "/").Body.String(), mine) {
		t.Error("the page is still wearing what it opened in")
	}

	// The file behind the name is read again as well.
	writeTheme(t, cfg, "sea.css", ":root{--numen-surface:#040506}")
	if !strings.Contains(getResponse(handler, "/").Body.String(), "#040506") {
		t.Error("the page is wearing the file as it was")
	}
}

// A build that cannot say what it wears serves the page as it was built, and
// the tokens the build carries stand.
func TestAPageThatCannotSayWhatItWearsIsServedAsItWasBuilt(t *testing.T) {
	files, err := Pages()
	if err != nil {
		t.Skipf("no interface in this binary: %v", err)
	}

	out := getResponse(newBareServices().NewHandler(files), "/")
	if out.Code != http.StatusOK {
		t.Fatalf("answered %d", out.Code)
	}
	if strings.Contains(out.Body.String(), "<style") {
		t.Error("the page carries a style element")
	}
}

// A theme naming the end of the element it is spliced into does not end it.
func TestAThemeCannotEndTheElementItIsIn(t *testing.T) {
	cfg := newConfig(t)
	handler, themes := newPageHandler(t, cfg)
	writeTheme(t, cfg, "loud.css", ":root{--numen-surface:#010203}</STYLE><b>out here</b>")
	choose(t, themes, "mine:loud", v1.Mode_MODE_SYSTEM)

	body := getResponse(handler, "/").Body.String()
	head, rest, found := strings.Cut(body, appearance.HeadEnd)
	if !found {
		t.Fatal("the page has no head")
	}
	if strings.Contains(rest, "out here") {
		t.Error("a theme wrote into the body")
	}
	if strings.Count(head, "</style>") != 3 {
		t.Errorf("the head holds %d style elements", strings.Count(head, "</style>"))
	}
	if !strings.Contains(head, `<\/STYLE>`) {
		t.Error("the theme's own text was not kept")
	}
}

// holdingOpen is a set of readers whose every look at a file waits until a test
// lets it through, which is what a question still inside its answer looks like
// from here.
type holdingOpen struct {
	port.VaultReaders
	begun chan struct{}
	until chan struct{}
}

func (h holdingOpen) Open(v domain.Vault) (port.VaultReader, error) {
	reader, err := h.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return holdsOpen{VaultReader: reader, at: h}, nil
}

type holdsOpen struct {
	port.VaultReader
	at holdingOpen
}

func (h holdsOpen) GetFingerprint(ctx context.Context, path string) (domain.Fingerprint, error) {
	select {
	case h.at.begun <- struct{}{}:
	default:
	}
	<-h.at.until
	return h.VaultReader.GetFingerprint(ctx, path)
}

// TestTheDoorShutsBehindTheQuestionsAlreadyTaken. A search, a note and a link
// are answered straight from the index, and one that passed the door a moment
// before it shut is still on the index when everything an answer reaches into
// is taken away.
func TestTheDoorShutsBehindTheQuestionsAlreadyTaken(t *testing.T) {
	readers := holdingOpen{
		VaultReaders: filesystem.VaultReaders{},
		begun:        make(chan struct{}, 1),
		until:        make(chan struct{}),
	}
	// A page is the one thing left under this route, and drawing one looks the
	// file up before anything else. That look is where the door stands.
	api := newViewingAPI(readers, newViewer(nil))
	api.Document.viewer.open = func(file io.ReadSeekCloser) (port.OpenDocument, error) {
		file.Close()
		return nil, document.ErrNoPage
	}
	api.state.setVault(testsupport.NewVault(t, map[string]string{"Note.pdf": "# Note\n"}))
	handler := api.NewHandler(http.NotFoundHandler())

	answered := make(chan struct{})
	go func() {
		defer close(answered)
		at := pageURL("Note.pdf", 0, 800, document.Fingerprint{})
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, at, nil))
	}()
	<-readers.begun

	shutCh := make(chan struct{})
	go func() {
		defer close(shutCh)
		api.state.Close()
	}()

	select {
	case <-shutCh:
		t.Fatal("the door shut while a question was still being answered")
	case <-time.After(100 * time.Millisecond):
	}

	close(readers.until)
	select {
	case <-shutCh:
	case <-time.After(10 * time.Second):
		t.Fatal("the door never shut")
	}
	<-answered
}

func TestPluginAssetsAreServed(t *testing.T) {
	pluginDir := t.TempDir()
	distDir := filepath.Join(pluginDir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}

	jsContent := []byte("console.log('plugin loaded');")
	cssContent := []byte(".plugin-widget { color: red; }")

	if err := os.WriteFile(filepath.Join(distDir, "index.js"), jsContent, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "styles.css"), cssContent, 0o644); err != nil {
		t.Fatal(err)
	}

	services := newBareServices()
	services.getPluginDir = func(id string) (string, bool) {
		if id == "sample.plugin" {
			return pluginDir, true
		}
		return "", false
	}
	handler := services.NewHandler(http.NotFoundHandler())

	t.Run("serves javascript module with correct mime", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/plugins/sample.plugin/dist/index.js", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/javascript" {
			t.Errorf("expected application/javascript, got %s", ct)
		}
		if w.Body.String() != string(jsContent) {
			t.Errorf("unexpected content: %s", w.Body.String())
		}
	})

	t.Run("serves css stylesheet with correct mime", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/plugins/sample.plugin/dist/styles.css", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if ct := w.Header().Get("Content-Type"); ct != "text/css" {
			t.Errorf("expected text/css, got %s", ct)
		}
		if w.Body.String() != string(cssContent) {
			t.Errorf("unexpected content: %s", w.Body.String())
		}
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/plugins/sample.plugin/../../secret", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusForbidden && w.Code != http.StatusBadRequest {
			t.Fatalf("expected 403 or 400, got %d", w.Code)
		}
	})

	t.Run("rejects symlink escape", func(t *testing.T) {
		outsideDir := t.TempDir()
		secretFile := filepath.Join(outsideDir, "secret.js")
		if err := os.WriteFile(secretFile, []byte("secret"), 0o644); err != nil {
			t.Fatal(err)
		}
		symlinkTarget := filepath.Join(distDir, "symlink.js")
		if err := os.Symlink(secretFile, symlinkTarget); err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodGet, "/api/plugins/sample.plugin/dist/symlink.js", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})

	t.Run("rejects an escaping directory symlink", func(t *testing.T) {
		outsideDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(outsideDir, "secret.js"), []byte("secret"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outsideDir, filepath.Join(pluginDir, "linkdir")); err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodGet, "/api/plugins/sample.plugin/linkdir/secret.js", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})

	t.Run("does not report a contained symlink to a file as an escape", func(t *testing.T) {
		if err := os.Symlink("index.js", filepath.Join(distDir, "alias.js")); err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodGet, "/api/plugins/sample.plugin/dist/alias.js/child", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code == http.StatusForbidden {
			t.Fatalf("contained symlink reported as an escape: %d", w.Code)
		}

		r = httptest.NewRequest(http.MethodGet, "/api/plugins/sample.plugin/dist/alias.js", nil)
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 through contained symlink, got %d", w.Code)
		}
	})

	t.Run("rejects a contained symlink that points at an escaping symlink", func(t *testing.T) {
		outsideDir := t.TempDir()
		if err := os.Symlink(outsideDir, filepath.Join(pluginDir, "outer")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("outer", filepath.Join(pluginDir, "inner")); err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodGet, "/api/plugins/sample.plugin/inner/secret.js", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})

	t.Run("answers 500 for a symlink loop", func(t *testing.T) {
		if err := os.Symlink("loopb", filepath.Join(pluginDir, "loopa")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("loopa", filepath.Join(pluginDir, "loopb")); err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodGet, "/api/plugins/sample.plugin/loopa/x.js", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("returns 404 for unknown plugin", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/plugins/unknown.plugin/dist/index.js", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("returns 404 for nonexistent asset", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/plugins/sample.plugin/dist/missing.js", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("returns 405 for POST method", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/api/plugins/sample.plugin/dist/index.js", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", w.Code)
		}
	})
}
