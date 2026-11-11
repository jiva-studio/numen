package editor

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	windowvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/vault"
	windowvaults "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/vaults"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/appearance"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/csp"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// mine is a user theme with a color no other theme has, pinning the dark
// scheme.
const mine = ":root{color-scheme:dark;--numen-surface:#010203}"

func newPageHandler(t *testing.T, cfg themeConfig) (http.Handler, numenv1connect.ThemeServiceHandler) {
	t.Helper()

	files, err := Pages()
	if err != nil {
		t.Skipf("no interface in this binary: %v", err)
	}
	themes, err := cfg.themes(nil)
	if err != nil {
		t.Fatal(err)
	}
	return newThemedServices(themes).NewHandler(files), themes
}

func writeTheme(t *testing.T, cfg themeConfig, name, body string) {
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

func writeAppearance(t *testing.T, themes numenv1connect.ThemeServiceHandler, requested *v1.WriteAppearanceRequest) {
	t.Helper()
	out, err := themes.WriteAppearance(t.Context(), connect.NewRequest(requested))
	if err != nil {
		t.Fatal(err)
	}
	if reason := out.Msg.GetError(); reason != "" {
		t.Fatalf("failed: %s", reason)
	}
}

func getResponse(tb testing.TB, handler http.Handler, path string) *httptest.ResponseRecorder {
	tb.Helper()
	r := httptest.NewRequestWithContext(tb.Context(), http.MethodGet, "/", nil)
	r.URL.Path = path
	out := httptest.NewRecorder()
	handler.ServeHTTP(out, r)
	return out
}

// TestHandler_AppliesOneContentSecurityPolicy checks what this window widens;
// the base policy is asserted in the csp package.
// A build that opens no socket widens nothing. A theme is a stylesheet that may
// have been downloaded and the policy is the only control that stops it
// reaching the network: a remote `url()` is blocked unless `style-src` permits its origin.
func TestHandler_AppliesOneContentSecurityPolicy(t *testing.T) {
	target := csp.Sources{}.Policy()

	handler := newBareServices().NewHandler(http.NotFoundHandler())
	for _, path := range []string{"", "/", "/index.html", "/built/index.css"} {
		if response := getResponse(t, handler, path).Header().Get("Content-Security-Policy"); response != target {
			t.Errorf("%q is bound to %q", path, response)
		}
	}
}

// The window frames no host. A frame runs that host's own scripts and calls
// that host's own machines, so any added host must appear in a reviewed diff.
func TestTheHostsAWindowMayFrame(t *testing.T) {
	response := getResponse(t, newBareServices().NewHandler(http.NotFoundHandler()), "/").
		Header().Get("Content-Security-Policy")
	if !strings.Contains(response, "frame-src 'self';") {
		t.Errorf("the policy reads %q", response)
	}
	if strings.Contains(response, "youtube") {
		t.Errorf("the window names a host of its own to frame: %q", response)
	}

	// A frame is what widens, and a script is not: what runs in the window is
	// what the window served, which is `default-src`.
	if strings.Contains(response, "script-src") {
		t.Errorf("the policy names where a script comes from: %q", response)
	}
}

// TestHandler_MountsEveryVaultService. Four services answer on the handler the
// application assembles. A route with nothing mounted falls through to the
// pages and returns not found; a route a service owns answers a GET with a
// method-not-allowed error.
func TestHandler_MountsEveryVaultService(t *testing.T) {
	services := newBareServices()
	vaultHandler := windowvault.NewHandler(services.state, services.state.Notices)
	vaultPath, vaultServed := windowvault.NewRoute(vaultHandler, services.state.Gate.CountQuestions())
	services.Extra = append(services.Extra, NewRoute(vaultPath, vaultServed, vaultHandler))
	handler := services.NewHandler(http.NotFoundHandler())
	for _, route := range []string{
		numenv1connect.VaultServiceGetVaultStateProcedure,
		numenv1connect.FileServiceListFilesProcedure,
		numenv1connect.NoteServiceReadNoteProcedure,
		numenv1connect.SearchServiceSearchNamesProcedure,
	} {
		if code := getResponse(t, handler, route).Code; code != http.StatusMethodNotAllowed {
			t.Errorf("%s answered %d, want %d", route, code, http.StatusMethodNotAllowed)
		}
	}
}

// TestPluginServiceIsMounted asserts that PluginServiceHandler is mounted
// behind the handler the application assembles.
func TestPluginServiceIsMounted(t *testing.T) {
	handler := newBareServices().NewHandler(http.NotFoundHandler())
	route := numenv1connect.PluginServiceGetPluginsProcedure
	if code := getResponse(t, handler, route).Code; code != http.StatusMethodNotAllowed {
		t.Errorf("%s answered %d, want %d", route, code, http.StatusMethodNotAllowed)
	}
}

// A search, a note and a link are read straight from the index, and the index
// closes behind the door. Every question fails at it, whatever it would
// have reached into.
func TestAWindowBeingTakenAwayAnswersNothing(t *testing.T) {
	services := newBareServices()
	handler := services.NewHandler(http.NotFoundHandler())
	questions := "/" + numenv1connect.VaultServiceName + "/"
	request := []string{"", "/", "/index.html", questions + "Find"}

	for _, path := range request {
		if code := getResponse(t, handler, path).Code; code == http.StatusServiceUnavailable {
			t.Errorf("%q failed at %d with the window still open", path, code)
		}
	}

	services.state.Gate.Close()

	for _, path := range request {
		if code := getResponse(t, handler, path).Code; code != http.StatusServiceUnavailable {
			t.Errorf("%q was answered %d by a window being taken away", path, code)
		}
	}
}

// The page is served with the theme applied at every address.
func TestThePageOpensWithTheTheme(t *testing.T) {
	cfg := newConfig(t)
	handler, themes := newPageHandler(t, cfg)
	writeTheme(t, cfg, "sea.css", mine)
	choose(t, themes, "mine:sea", v1.Mode_MODE_DARK)

	for _, path := range []string{"", "/", "/index.html"} {
		t.Run(strconv.Quote(path), func(t *testing.T) {
			out := getResponse(t, handler, path)
			if out.Code != http.StatusOK {
				t.Fatalf("answered %d", out.Code)
			}
			if response := out.Header().Get("Content-Type"); !strings.HasPrefix(response, "text/html") {
				t.Errorf("came back as %q", response)
			}
			if !strings.Contains(out.Body.String(), mine) {
				t.Error("the page does not apply the theme")
			}
			if responseVal := out.Header().Get("Content-Length"); responseVal != strconv.Itoa(out.Body.Len()) {
				t.Errorf("%s bytes were announced and %d written", responseVal, out.Body.Len())
			}
		})
	}
}

// TestTheStyleElementsAreTheLastThingInTheHead checks that the mode's element, the theme's and the
// two sizes are the last three things in the head, in that order.
// A theme and the built stylesheet declare `color-scheme` with equal weight, so
// the later wins: the theme follows the built stylesheet's link and the mode,
// so a theme pinning the scheme takes effect. The sizes follow the theme.
func TestTheStyleElementsAreTheLastThingInTheHead(t *testing.T) {
	cfg := newConfig(t)
	handler, themes := newPageHandler(t, cfg)
	writeTheme(t, cfg, "sea.css", mine)
	writeAppearance(t, themes, &v1.WriteAppearanceRequest{
		Name:           "mine:sea",
		Mode:           v1.Mode_MODE_DARK,
		InterfaceScale: new(1.25),
		TextScale:      new(1.5),
	})

	head, _, found := strings.Cut(getResponse(t, handler, "/").Body.String(), appearance.HeadEnd)
	if !found {
		t.Fatal("the page has no head")
	}

	const rendered = `<style data-appearance="sizes">` +
		":root { --numen-interface-scale: 1.25; --numen-text-scale: 1.5; }</style>"
	link := strings.LastIndex(head, "<link")
	mode := strings.Index(head,
		`<style data-appearance="mode">`+":root { color-scheme: dark; }</style>")
	mineAt := strings.Index(head, mine)
	sizes := strings.Index(head, rendered)
	if link < 0 || mode < 0 || mineAt < 0 || sizes < 0 {
		t.Fatalf("the link is at %d, the mode at %d, the theme at %d, the sizes at %d",
			link, mode, mineAt, sizes)
	}
	if link > mode || mode > mineAt || mineAt > sizes {
		t.Errorf("the link is at %d, the mode at %d, the theme at %d, the sizes at %d",
			link, mode, mineAt, sizes)
	}
	if after := strings.TrimSpace(head[sizes+len(rendered):]); after != "" {
		t.Errorf("the head ends with %q", after)
	}
}

// The page is drawn at the interface scale the file names.
func TestPageUsesInterfaceScaleOfFile(t *testing.T) {
	cfg := newConfig(t)
	file := filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json")
	if err := os.WriteFile(file, []byte(`{"app":{"appearance":{"interfaceScale":1.5}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	handler, _ := newPageHandler(t, cfg)

	if !strings.Contains(getResponse(t, handler, "/").Body.String(), "--numen-interface-scale: 1.5") {
		t.Error("the page is not drawn at what the file says")
	}
}

// The built stylesheet declares `color-scheme` at zero weight alone, so the
// element the page carries takes effect regardless of the theme.
func TestTheBuiltStylesheetDoesNotPinTheColorScheme(t *testing.T) {
	constructed, err := wire.GetInterface(pages)
	if err != nil {
		t.Skipf("no interface in this binary: %v", err)
	}

	sheets, err := fs.Glob(constructed, "built/*.css")
	if err != nil || len(sheets) == 0 {
		t.Fatalf("the build has no stylesheet: %v", err)
	}
	var declaresScheme bool
	for _, name := range sheets {
		text, err := fs.ReadFile(constructed, name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(text), ":root{color-scheme") {
			t.Errorf("%s declares the color scheme at the root", name)
		}
		if regexp.MustCompile(`:where\(:root\)\s*\{[^}]*color-scheme`).Match(text) {
			declaresScheme = true
		}
	}
	if !declaresScheme {
		t.Error("the built stylesheets declare no color scheme at all")
	}
}

// The head is read for each request: a newly chosen theme applies on the next
// reload.
func TestTheHeadIsReadForEachRequest(t *testing.T) {
	cfg := newConfig(t)
	handler, themes := newPageHandler(t, cfg)
	writeTheme(t, cfg, "sea.css", mine)

	if strings.Contains(getResponse(t, handler, "/").Body.String(), mine) {
		t.Fatal("the page applies a theme nobody chose")
	}
	choose(t, themes, "mine:sea", v1.Mode_MODE_DARK)
	if !strings.Contains(getResponse(t, handler, "/").Body.String(), mine) {
		t.Error("the page still applies the theme it opened with")
	}

	// The file behind the name is read again as well.
	writeTheme(t, cfg, "sea.css", ":root{--numen-surface:#040506}")
	if !strings.Contains(getResponse(t, handler, "/").Body.String(), "#040506") {
		t.Error("the page applies the file as it was")
	}
}

// A build that cannot report its theme serves the page as built, and the
// build's tokens apply.
func TestPages_ServesUnthemedPageUnchanged(t *testing.T) {
	files, err := Pages()
	if err != nil {
		t.Skipf("no interface in this binary: %v", err)
	}

	out := getResponse(t, newBareServices().NewHandler(files), "/")
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

	body := getResponse(t, handler, "/").Body.String()
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

type blockingVaults struct {
	begun chan struct{}
	until chan struct{}
}

func (b blockingVaults) List() ([]domain.Vault, error) {
	select {
	case b.begun <- struct{}{}:
	default:
	}
	<-b.until
	return nil, nil
}

// TestHandler_ClosingWaitsForInFlightRequests. A search, a note and a link are
// answered from the index. A request admitted immediately before the handler closes
// still uses the index while everything its answer depends on is torn down.
func TestHandler_ClosingWaitsForInFlightRequests(t *testing.T) {
	begun := make(chan struct{}, 1)
	until := make(chan struct{})
	b := blockingVaults{begun: begun, until: until}
	list := registry.NewList(b)
	check := registry.NewFolderCheck(vaultreader.Readers{})
	api := newBareServices()
	vaultsHandler := windowvaults.NewHandler(api.state, windowvaults.Uses{List: list, FolderCheck: check})
	vaultsPath, vaultsServed := windowvaults.NewRoute(vaultsHandler, api.state.Gate.CountQuestions())
	api.Extra = append(api.Extra, NewRoute(vaultsPath, vaultsServed, vaultsHandler))
	handler := api.NewHandler(http.NotFoundHandler())

	reply := make(chan struct{})
	go func() {
		defer close(reply)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, vaultsPath+"ListVaults", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}()
	<-begun

	shutCh := make(chan struct{})
	go func() {
		defer close(shutCh)
		api.state.Gate.Close()
	}()

	select {
	case <-shutCh:
		t.Fatal("the door shut while a question was still being answered")
	case <-time.After(100 * time.Millisecond):
	}

	close(until)
	select {
	case <-shutCh:
	case <-time.After(10 * time.Second):
		t.Fatal("the door never shut")
	}
	<-reply
}

func TestPluginAssetsAreServed(t *testing.T) {
	fixture := makePluginFixture(t)

	t.Run("serves javascript module with correct mime", func(t *testing.T) {
		w := fixture.serveRequest(t, http.MethodGet, "/plugin/sample.plugin/dist/index.js")
		verifyAssetResponse(t, w, http.StatusOK, "application/javascript", string(fixture.jsContent))
	})

	t.Run("serves css stylesheet with correct mime", func(t *testing.T) {
		w := fixture.serveRequest(t, http.MethodGet, "/plugin/sample.plugin/dist/styles.css")
		verifyAssetResponse(t, w, http.StatusOK, "text/css", string(fixture.cssContent))
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		w := fixture.serveRequest(t, http.MethodGet, "/plugin/sample.plugin/../../secret")
		if w.Code != http.StatusForbidden && w.Code != http.StatusBadRequest {
			t.Fatalf("expected 403 or 400, got %d", w.Code)
		}
	})

	t.Run("rejects symlink escape", func(t *testing.T) {
		fixture.createSymlinkEscape(t)
		w := fixture.serveRequest(t, http.MethodGet, "/plugin/sample.plugin/dist/symlink.js")
		verifyAssetResponse(t, w, http.StatusForbidden, "", "")
	})

	t.Run("rejects an escaping directory symlink", func(t *testing.T) {
		fixture.createDirSymlinkEscape(t)
		w := fixture.serveRequest(t, http.MethodGet, "/plugin/sample.plugin/linkdir/secret.js")
		verifyAssetResponse(t, w, http.StatusForbidden, "", "")
	})

	t.Run("does not report a contained symlink to a file as an escape", func(t *testing.T) {
		fixture.createContainedSymlink(t)
		w := fixture.serveRequest(t, http.MethodGet, "/plugin/sample.plugin/dist/alias.js/child")
		if w.Code == http.StatusForbidden {
			t.Fatalf("contained symlink reported as an escape: %d", w.Code)
		}

		w = fixture.serveRequest(t, http.MethodGet, "/plugin/sample.plugin/dist/alias.js")
		verifyAssetResponse(t, w, http.StatusOK, "", "")
	})

	t.Run("rejects a contained symlink that points at an escaping symlink", func(t *testing.T) {
		fixture.createChainedSymlinkEscape(t)
		w := fixture.serveRequest(t, http.MethodGet, "/plugin/sample.plugin/inner/secret.js")
		verifyAssetResponse(t, w, http.StatusForbidden, "", "")
	})

	t.Run("answers 500 for a symlink loop", func(t *testing.T) {
		fixture.createSymlinkLoop(t)
		w := fixture.serveRequest(t, http.MethodGet, "/plugin/sample.plugin/loopa/x.js")
		verifyAssetResponse(t, w, http.StatusInternalServerError, "", "")
	})

	t.Run("returns 404 for unknown plugin", func(t *testing.T) {
		w := fixture.serveRequest(t, http.MethodGet, "/plugin/unknown.plugin/dist/index.js")
		verifyAssetResponse(t, w, http.StatusNotFound, "", "")
	})

	t.Run("returns 404 for nonexistent asset", func(t *testing.T) {
		w := fixture.serveRequest(t, http.MethodGet, "/plugin/sample.plugin/dist/missing.js")
		verifyAssetResponse(t, w, http.StatusNotFound, "", "")
	})

	t.Run("returns 405 for POST method", func(t *testing.T) {
		w := fixture.serveRequest(t, http.MethodPost, "/plugin/sample.plugin/dist/index.js")
		verifyAssetResponse(t, w, http.StatusMethodNotAllowed, "", "")
	})
}

type pluginFixture struct {
	pluginDir  string
	distDir    string
	handler    http.Handler
	jsContent  []byte
	cssContent []byte
}

func makePluginFixture(t *testing.T) *pluginFixture {
	t.Helper()
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

	return &pluginFixture{
		pluginDir:  pluginDir,
		distDir:    distDir,
		handler:    handler,
		jsContent:  jsContent,
		cssContent: cssContent,
	}
}

func (f *pluginFixture) serveRequest(tb testing.TB, method, path string) *httptest.ResponseRecorder {
	tb.Helper()
	r := httptest.NewRequestWithContext(tb.Context(), method, path, nil)
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w
}

func (f *pluginFixture) createSymlinkEscape(t *testing.T) {
	t.Helper()
	outsideDir := t.TempDir()
	secretFile := filepath.Join(outsideDir, "secret.js")
	if err := os.WriteFile(secretFile, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	symlinkTarget := filepath.Join(f.distDir, "symlink.js")
	if err := os.Symlink(secretFile, symlinkTarget); err != nil {
		t.Fatal(err)
	}
}

func (f *pluginFixture) createDirSymlinkEscape(t *testing.T) {
	t.Helper()
	outsideDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outsideDir, "secret.js"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(f.pluginDir, "linkdir")); err != nil {
		t.Fatal(err)
	}
}

func (f *pluginFixture) createContainedSymlink(t *testing.T) {
	t.Helper()
	if err := os.Symlink("index.js", filepath.Join(f.distDir, "alias.js")); err != nil {
		t.Fatal(err)
	}
}

func (f *pluginFixture) createChainedSymlinkEscape(t *testing.T) {
	t.Helper()
	outsideDir := t.TempDir()
	if err := os.Symlink(outsideDir, filepath.Join(f.pluginDir, "outer")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("outer", filepath.Join(f.pluginDir, "inner")); err != nil {
		t.Fatal(err)
	}
}

func (f *pluginFixture) createSymlinkLoop(t *testing.T) {
	t.Helper()
	if err := os.Symlink("loopb", filepath.Join(f.pluginDir, "loopa")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("loopa", filepath.Join(f.pluginDir, "loopb")); err != nil {
		t.Fatal(err)
	}
}

func verifyAssetResponse(t *testing.T, w *httptest.ResponseRecorder, wantStatus int, wantType, wantBody string) {
	t.Helper()
	if w.Code != wantStatus {
		t.Fatalf("expected %d, got %d", wantStatus, w.Code)
	}
	if wantType != "" {
		if ct := w.Header().Get("Content-Type"); ct != wantType {
			t.Errorf("expected %s, got %s", wantType, ct)
		}
	}
	if wantBody != "" && w.Body.String() != wantBody {
		t.Errorf("unexpected content: %s", w.Body.String())
	}
}
