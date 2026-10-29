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

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/internal/appearance"
	"github.com/jiva-studio/numen/libs/core/internal/csp"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
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

// size is a size a choice names.
func size(response float64) *float64 { return &response }

// getResponse is what comes back at exactly this path. The page is handed over as
// the empty path as well, which no URL parses to.
func getResponse(handler http.Handler, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.URL.Path = path
	out := httptest.NewRecorder()
	handler.ServeHTTP(out, r)
	return out
}

// to say and is asserted there; what this window widens is asserted here.
//
// A build that opened no socket plays from nowhere, so it widens nothing. A
// theme is an ordinary stylesheet a person may have downloaded, and the policy
// is the whole of what stops one reaching the network: a remote `url()` is
// into is permitted by `style-src`.
func TestHandler_AppliesOneContentSecurityPolicy(t *testing.T) {
	target := csp.Sources{}.Policy()

	handler := newBareServices().NewHandler(http.NotFoundHandler())
	for _, path := range []string{"", "/", "/index.html", "/built/index.css"} {
		if response := getResponse(handler, path).Header().Get("Content-Security-Policy"); response != target {
			t.Errorf("%q is bound to %q", path, response)
		}
	}
}

// The window frames no host. A frame runs that host's own scripts and reaches
// that host's own machines, so a host arriving here arrives in a diff somebody
// reads.
func TestTheHostsAWindowMayFrame(t *testing.T) {
	response := getResponse(newBareServices().NewHandler(http.NotFoundHandler()), "/").
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

// TestHandler_MountsEveryVaultService. Four services answer about the
// handler the application hands over. A route nothing is mounted at falls
// through to the pages and is answered not found; a route a service holds
// answers a GET as a method that call does not take.
func TestHandler_MountsEveryVaultService(t *testing.T) {
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
	request := []string{"", "/", "/index.html", questions + "Find"}

	for _, path := range request {
		if code := getResponse(handler, path).Code; code == http.StatusServiceUnavailable {
			t.Errorf("%q failed at %d with the window still open", path, code)
		}
	}

	services.state.Close()

	for _, path := range request {
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
			if response := out.Header().Get("Content-Type"); !strings.HasPrefix(response, "text/html") {
				t.Errorf("came back as %q", response)
			}
			if !strings.Contains(out.Body.String(), mine) {
				t.Error("the page is not wearing the theme")
			}
			if responseVal := out.Header().Get("Content-Length"); responseVal != strconv.Itoa(out.Body.Len()) {
				t.Errorf("%s bytes were announced and %d written", responseVal, out.Body.Len())
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

	const rendered = `<style data-appearance="sizes">` +
		":root { --numen-interface-scale: 1.25; --numen-text-scale: 1.5; }</style>"
	link := strings.LastIndex(head, "<link")
	mode := strings.Index(head,
		`<style data-appearance="mode">`+":root { color-scheme: dark; }</style>")
	aged := strings.Index(head, mine)
	sizes := strings.Index(head, rendered)
	if link < 0 || mode < 0 || aged < 0 || sizes < 0 {
		t.Fatalf("the link is at %d, the mode at %d, the theme at %d, the sizes at %d",
			link, mode, aged, sizes)
	}
	if link > mode || mode > aged || aged > sizes {
		t.Errorf("the link is at %d, the mode at %d, the theme at %d, the sizes at %d",
			link, mode, aged, sizes)
	}
	if after := strings.TrimSpace(head[sizes+len(rendered):]); after != "" {
		t.Errorf("the head ends with %q", after)
	}
}

// A window drawn at 1.5 goes on being drawn at 1.5.
func TestAFileNamingTheZoomOpensTheWindowGeneratedAtIt(t *testing.T) {
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
func TestPages_ServesUnthemedPageUnchanged(t *testing.T) {
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

type blockingVaults struct {
	vaultport.VaultRegistry
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

func (blockingVaults) FindVault(string) (domain.Vault, bool, error) {
	return domain.Vault{}, false, nil
}
func (blockingVaults) GetLastVault() (domain.Vault, bool, error) { return domain.Vault{}, false, nil }
func (blockingVaults) Save(domain.Vault) error                   { return nil }
func (blockingVaults) Remove(domain.VaultID) error               { return nil }
func (blockingVaults) RecordOpened(domain.VaultID) error         { return nil }

// TestHandler_ClosingWaitsForInFlightRequests. A search, a note and a link
// are answered straight from the index, and one that passed the door a moment
// before it shut is still on the index when everything an answer reaches into
// is taken away.
func TestHandler_ClosingWaitsForInFlightRequests(t *testing.T) {
	begun := make(chan struct{}, 1)
	until := make(chan struct{})
	b := blockingVaults{begun: begun, until: until}
	list := vaults.NewList(b)
	check := vaults.NewFolderCheck(vaultreader.Readers{})
	api := newBareServices()
	api.Vaults.list = &list
	api.Vaults.folderCheck = &check
	handler := api.NewHandler(http.NotFoundHandler())

	reply := make(chan struct{})
	go func() {
		defer close(reply)
		path, _ := numenv1connect.NewVaultsServiceHandler(api.Vaults)
		req := httptest.NewRequest(http.MethodPost, path+"ListVaults", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}()
	<-begun

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
		w := fixture.serveRequest(http.MethodGet, "/plugin/sample.plugin/dist/index.js")
		verifyAssetResponse(t, w, http.StatusOK, "application/javascript", string(fixture.jsContent))
	})

	t.Run("serves css stylesheet with correct mime", func(t *testing.T) {
		w := fixture.serveRequest(http.MethodGet, "/plugin/sample.plugin/dist/styles.css")
		verifyAssetResponse(t, w, http.StatusOK, "text/css", string(fixture.cssContent))
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		w := fixture.serveRequest(http.MethodGet, "/plugin/sample.plugin/../../secret")
		if w.Code != http.StatusForbidden && w.Code != http.StatusBadRequest {
			t.Fatalf("expected 403 or 400, got %d", w.Code)
		}
	})

	t.Run("rejects symlink escape", func(t *testing.T) {
		fixture.createSymlinkEscape(t)
		w := fixture.serveRequest(http.MethodGet, "/plugin/sample.plugin/dist/symlink.js")
		verifyAssetResponse(t, w, http.StatusForbidden, "", "")
	})

	t.Run("rejects an escaping directory symlink", func(t *testing.T) {
		fixture.createDirSymlinkEscape(t)
		w := fixture.serveRequest(http.MethodGet, "/plugin/sample.plugin/linkdir/secret.js")
		verifyAssetResponse(t, w, http.StatusForbidden, "", "")
	})

	t.Run("does not report a contained symlink to a file as an escape", func(t *testing.T) {
		fixture.createContainedSymlink(t)
		w := fixture.serveRequest(http.MethodGet, "/plugin/sample.plugin/dist/alias.js/child")
		if w.Code == http.StatusForbidden {
			t.Fatalf("contained symlink reported as an escape: %d", w.Code)
		}

		w = fixture.serveRequest(http.MethodGet, "/plugin/sample.plugin/dist/alias.js")
		verifyAssetResponse(t, w, http.StatusOK, "", "")
	})

	t.Run("rejects a contained symlink that points at an escaping symlink", func(t *testing.T) {
		fixture.createChainedSymlinkEscape(t)
		w := fixture.serveRequest(http.MethodGet, "/plugin/sample.plugin/inner/secret.js")
		verifyAssetResponse(t, w, http.StatusForbidden, "", "")
	})

	t.Run("answers 500 for a symlink loop", func(t *testing.T) {
		fixture.createSymlinkLoop(t)
		w := fixture.serveRequest(http.MethodGet, "/plugin/sample.plugin/loopa/x.js")
		verifyAssetResponse(t, w, http.StatusInternalServerError, "", "")
	})

	t.Run("returns 404 for unknown plugin", func(t *testing.T) {
		w := fixture.serveRequest(http.MethodGet, "/plugin/unknown.plugin/dist/index.js")
		verifyAssetResponse(t, w, http.StatusNotFound, "", "")
	})

	t.Run("returns 404 for nonexistent asset", func(t *testing.T) {
		w := fixture.serveRequest(http.MethodGet, "/plugin/sample.plugin/dist/missing.js")
		verifyAssetResponse(t, w, http.StatusNotFound, "", "")
	})

	t.Run("returns 405 for POST method", func(t *testing.T) {
		w := fixture.serveRequest(http.MethodPost, "/plugin/sample.plugin/dist/index.js")
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

func (f *pluginFixture) serveRequest(method, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, nil)
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
