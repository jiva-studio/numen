package webui

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/appearance"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// mine is a theme of the person's, carrying a colour nothing else has and
// pinning the dark half.
const mine = ":root{color-scheme:dark;--numen-surface:#010203}"

// drawn is the whole handler as the application hands it over: the questions,
// the themes, and the built page behind them.
func drawn(t *testing.T, cfg container.Config) (http.Handler, numenv1connect.ThemeServiceHandler) {
	t.Helper()

	files, err := Pages()
	if err != nil {
		t.Skipf("no interface in this binary: %v", err)
	}
	themes, err := cfg.Themes(nil)
	if err != nil {
		t.Fatal(err)
	}
	return (&API{Themes: themes}).Serving(files), themes
}

// puts a theme in the person's folder, which opening the catalogue made.
func puts(t *testing.T, cfg container.Config, name, body string) {
	t.Helper()
	folder := filepath.Join(filepath.Dir(cfg.RegistryPath), "themes")
	if err := os.WriteFile(filepath.Join(folder, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func choose(t *testing.T, themes numenv1connect.ThemeServiceHandler, name string, mode v1.Mode) {
	t.Helper()
	chose(t, themes, &v1.ChooseRequest{Name: name, Mode: mode})
}

// chose is one choice as a client makes it, whatever of it the client names.
func chose(t *testing.T, themes numenv1connect.ThemeServiceHandler, asked *v1.ChooseRequest) {
	t.Helper()
	out, err := themes.Choose(t.Context(), connect.NewRequest(asked))
	if err != nil {
		t.Fatal(err)
	}
	if failed := out.Msg.GetFailed(); failed != "" {
		t.Fatalf("refused: %s", failed)
	}
}

// size is a size a choice names.
func size(said float64) *float64 { return &said }

// handed is what comes back at exactly this path. The page is handed over as
// the empty path as well, which no URL parses to.
func handed(handler http.Handler, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.URL.Path = path
	out := httptest.NewRecorder()
	handler.ServeHTTP(out, r)
	return out
}

// The policy the window is held to, written out here as well as in the handler.
//
// A theme is an ordinary stylesheet a person may have downloaded, and this line
// is the whole of what stops one reaching the network: a remote `url()` is
// refused by `img-src` and `font-src`, and the element the theme is spliced
// into is permitted by `style-src`.
func TestTheWindowIsHeldToOnePolicy(t *testing.T) {
	const held = "default-src 'self'; img-src 'self'; media-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; " +
		"form-action 'none'; frame-ancestors 'none'"

	handler := (&API{}).Serving(http.NotFoundHandler())
	for _, path := range []string{"", "/", "/index.html", "/built/index.css", assetOf("a.pdf")} {
		if said := handed(handler, path).Header().Get("Content-Security-Policy"); said != held {
			t.Errorf("%q is held to %q", path, said)
		}
	}
}

// TestEveryServiceTheVaultIsAskedAboutIsMounted. Four services answer about the
// vault a window is showing, and every one of them has to be behind the one
// handler the application hands over. A route nothing is mounted at falls
// through to the pages and is answered not found; a route a service holds
// answers a GET as a method that call does not take.
func TestEveryServiceTheVaultIsAskedAboutIsMounted(t *testing.T) {
	handler := (&API{}).Serving(http.NotFoundHandler())
	for _, route := range []string{
		numenv1connect.VaultServiceStateProcedure,
		numenv1connect.FileServiceListProcedure,
		numenv1connect.NoteServiceReadProcedure,
		numenv1connect.SearchServiceNamesProcedure,
	} {
		if code := handed(handler, route).Code; code != http.StatusMethodNotAllowed {
			t.Errorf("%s answered %d, want %d", route, code, http.StatusMethodNotAllowed)
		}
	}
}

// A search, a note and a link are read straight from the index, and the index
// closes behind the door. Every question is refused at it, whatever it would
// have reached into.
func TestAWindowBeingTakenAwayAnswersNothing(t *testing.T) {
	api := &API{}
	handler := api.Serving(http.NotFoundHandler())
	questions, _ := numenv1connect.NewVaultServiceHandler(api)
	asked := []string{"", "/", "/index.html", assetOf("a.pdf"), questions + "Find"}

	for _, path := range asked {
		if code := handed(handler, path).Code; code == http.StatusServiceUnavailable {
			t.Errorf("%q was refused at %d with the window still open", path, code)
		}
	}

	api.Shut()

	for _, path := range asked {
		if code := handed(handler, path).Code; code != http.StatusServiceUnavailable {
			t.Errorf("%q was answered %d by a window being taken away", path, code)
		}
	}
}

// The page arrives wearing the theme, at every address it is asked for under.
func TestThePageOpensWearingTheTheme(t *testing.T) {
	cfg := installed(t)
	handler, themes := drawn(t, cfg)
	puts(t, cfg, "sea.css", mine)
	choose(t, themes, "mine:sea", v1.Mode_MODE_DARK)

	for _, path := range []string{"", "/", "/index.html"} {
		t.Run(strconv.Quote(path), func(t *testing.T) {
			out := handed(handler, path)
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
	cfg := installed(t)
	handler, themes := drawn(t, cfg)
	puts(t, cfg, "sea.css", mine)
	chose(t, themes, &v1.ChooseRequest{
		Name:           "mine:sea",
		Mode:           v1.Mode_MODE_DARK,
		InterfaceScale: size(1.25),
		TextScale:      size(1.5),
	})

	head, _, found := strings.Cut(handed(handler, "/").Body.String(), appearance.HeadEnd)
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
	cfg := installed(t)
	file := filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json")
	if err := os.WriteFile(file, []byte(`{"appearance":{"zoom":1.5}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	handler, _ := drawn(t, cfg)

	if !strings.Contains(handed(handler, "/").Body.String(), "--numen-interface-scale: 1.5") {
		t.Error("the page is not drawn at what the file says")
	}
}

// The built stylesheet declares `color-scheme` at zero weight alone, so the
// element the page carries stands unopposed whatever a theme says.
func TestTheBuiltStylesheetDoesNotPinTheColourScheme(t *testing.T) {
	built, err := appearance.Built(pages)
	if err != nil {
		t.Skipf("no interface in this binary: %v", err)
	}

	sheets, err := fs.Glob(built, "built/*.css")
	if err != nil || len(sheets) == 0 {
		t.Fatalf("the build has no stylesheet: %v", err)
	}
	for _, name := range sheets {
		text, err := fs.ReadFile(built, name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(text), ":root{color-scheme") {
			t.Errorf("%s declares the colour scheme at the root", name)
		}
		if !strings.Contains(string(text), ":where(:root){color-scheme") {
			t.Errorf("%s declares no colour scheme at all", name)
		}
	}
}

// The head is read for each request: a theme chosen is worn by the next reload.
func TestTheHeadIsReadForEachRequest(t *testing.T) {
	cfg := installed(t)
	handler, themes := drawn(t, cfg)
	puts(t, cfg, "sea.css", mine)

	if strings.Contains(handed(handler, "/").Body.String(), mine) {
		t.Fatal("the page is wearing a theme nobody chose")
	}
	choose(t, themes, "mine:sea", v1.Mode_MODE_DARK)
	if !strings.Contains(handed(handler, "/").Body.String(), mine) {
		t.Error("the page is still wearing what it opened in")
	}

	// The file behind the name is read again as well.
	puts(t, cfg, "sea.css", ":root{--numen-surface:#040506}")
	if !strings.Contains(handed(handler, "/").Body.String(), "#040506") {
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

	out := handed((&API{}).Serving(files), "/")
	if out.Code != http.StatusOK {
		t.Fatalf("answered %d", out.Code)
	}
	if strings.Contains(out.Body.String(), "<style") {
		t.Error("the page carries a style element")
	}
}

// A theme naming the end of the element it is spliced into does not end it.
func TestAThemeCannotEndTheElementItIsIn(t *testing.T) {
	cfg := installed(t)
	handler, themes := drawn(t, cfg)
	puts(t, cfg, "loud.css", ":root{--numen-surface:#010203}</STYLE><b>out here</b>")
	choose(t, themes, "mine:loud", v1.Mode_MODE_SYSTEM)

	body := handed(handler, "/").Body.String()
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

func (h holdsOpen) Stat(ctx context.Context, path string) (domain.Fingerprint, error) {
	select {
	case h.at.begun <- struct{}{}:
	default:
	}
	<-h.at.until
	return h.VaultReader.Stat(ctx, path)
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
	// file up before anything else. That look is where the door stands; what
	// the page would have been drawn from is never opened.
	api := &API{Readers: readers, Viewer: looking(nil)}
	api.Viewer.open = func([]byte) (scan, error) { return nil, errNoDrawing }
	api.show(testsupport.NewVault(t, map[string]string{"Note.md": "# Note\n"}))
	handler := api.Serving(http.NotFoundHandler())

	answered := make(chan struct{})
	go func() {
		defer close(answered)
		at := pageOf("Note.md", 0, 800)
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, at, nil))
	}()
	<-readers.begun

	shut := make(chan struct{})
	go func() {
		defer close(shut)
		api.Shut()
	}()

	select {
	case <-shut:
		t.Fatal("the door shut while a question was still being answered")
	case <-time.After(100 * time.Millisecond):
	}

	close(readers.until)
	select {
	case <-shut:
	case <-time.After(10 * time.Second):
		t.Fatal("the door never shut")
	}
	<-answered
}
