package webui

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/pdf"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
)

// Where the document a test asks about sits in its vault, and a file beside it
// that the vault holds as nothing.
const (
	book    = "library/scan.pdf"
	another = "library/other.pdf"
	beside  = "library/beside.txt"
)

// paper is a document a test draws from: pages of a size it names, counting
// every drawing and every opening, so a page drawn twice is visible.
type paper struct {
	mu     sync.Mutex
	opens  int
	draws  int
	closes int
	// each counts the drawings of one page, which a total cannot tell apart
	// from the drawing of another.
	each map[int]int

	pages int
	// wide and high are what every page measures in its own units.
	wide, high int

	// gate stops an opening where a test can see it, and is nil for a document
	// that opens at once.
	gate chan struct{}
}

func sheets(pages int) *paper {
	return &paper{pages: pages, wide: 612, high: 792}
}

// opened is what the viewer is given in place of pdf.Open.
func (p *paper) opened([]byte) (scan, error) {
	if p.gate != nil {
		<-p.gate
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.opens++
	return p, nil
}

func (p *paper) Pages() int { return p.pages }

// Label numbers the first two pages as front matter, the way a book does.
// Size is what the page measures in its own units, which the document answers
// without drawing anything.
func (p *paper) Size(int) (wide, high float64, err error) {
	return float64(p.wide), float64(p.high), nil
}

func (p *paper) Image(index, dpi int) (image.Image, error) {
	p.mu.Lock()
	p.draws++
	if p.each == nil {
		p.each = map[int]int{}
	}
	p.each[index]++
	p.mu.Unlock()

	drawn := image.NewRGBA(image.Rect(0, 0, p.wide*dpi/pointsDPI, p.high*dpi/pointsDPI))
	for y := drawn.Bounds().Min.Y; y < drawn.Bounds().Max.Y; y++ {
		for x := drawn.Bounds().Min.X; x < drawn.Bounds().Max.X; x++ {
			drawn.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: uint8(index), A: 255})
		}
	}
	return drawn, nil
}

func (p *paper) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closes++
}

func (p *paper) counted() (opens, draws, closes int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.opens, p.draws, p.closes
}

// drewPage is how many times one page was drawn.
func (p *paper) drewPage(index int) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.each[index]
}

// drawnFrom is a window looking at one document, with the pages drawn by the
// paper it is given.
func drawnFrom(t *testing.T, from *paper) (*API, http.Handler) {
	t.Helper()
	vault := testsupport.NewVault(t, map[string]string{
		book:    "the bytes of a scan",
		another: "the bytes of a second scan",
		beside:  "a file the vault leaves alone",
	})
	api := &API{Readers: filesystem.VaultReaders{}, Viewer: looking(pdf.Documents{})}
	api.show(vault)
	api.Viewer.open = from.opened
	t.Cleanup(api.Viewer.close)
	return api, api.Serving(http.NotFoundHandler())
}

// alone turns off the page drawn ahead, for a test counting what was drawn: a
// slot with no room in it is a drawing that never starts.
func alone(api *API) { api.Viewer.ahead.reading = make(chan struct{}) }

// fromTheLibrary is a window looking at a file the library itself draws.
//
// It is given longer than a page turn is: the library compiles itself the first
// time anything asks it for a document.
func fromTheLibrary(t *testing.T, raw string) (*API, http.Handler) {
	t.Helper()
	vault := testsupport.NewVault(t, map[string]string{book: raw})
	api := &API{Readers: filesystem.VaultReaders{}, Viewer: looking(pdf.Documents{})}
	api.show(vault)
	api.Viewer.patience = time.Minute
	t.Cleanup(api.Viewer.close)
	return api, api.Serving(http.NotFoundHandler())
}

// ask puts one request to the window and answers with what came back.
func ask(handler http.Handler, url string) *httptest.ResponseRecorder {
	out := httptest.NewRecorder()
	handler.ServeHTTP(out, httptest.NewRequest(http.MethodGet, url, nil))
	return out
}

// A page comes back drawn at least as wide as the window asked for it, and
// keeping its shape. A resolution is a whole number, so the width lands a pixel
// or two over; the window lays the page out at the width it asked for, and a
// page narrower than that would be laid out blurred.
func TestAPageComesBackDrawnAsWideAsWasAsked(t *testing.T) {
	for _, wide := range []int{200, 612, 1000, 1} {
		t.Run(fmt.Sprint(wide), func(t *testing.T) {
			api, handler := drawnFrom(t, sheets(4))
			alone(api)

			out := ask(handler, pageOf(book, 0, wide))
			if out.Code != http.StatusOK {
				t.Fatalf("asked for a page and got %d: %s", out.Code, out.Body)
			}
			if said := out.Header().Get("Content-Type"); said != "image/jpeg" {
				t.Errorf("the page came back as %q", said)
			}
			drawn, err := jpeg.Decode(out.Body)
			if err != nil {
				t.Fatalf("what came back is not a picture: %v", err)
			}
			got := drawn.Bounds().Dx()
			if got < wide {
				t.Errorf("asked for %d pixels wide and got %d", wide, got)
			}
			// The page keeps its shape: 612 by 792 units of its own.
			high := drawn.Bounds().Dy()
			if want := got * 792 / 612; high < want-2 || high > want+2 {
				t.Errorf("a page %d wide came back %d high, not about %d", got, high, want)
			}
		})
	}
}

// What a document is: how many pages it has, and what a person reading it would
// call each one.
func TestWhatADocumentIsIsHowManyPagesAndWhatEachIsCalled(t *testing.T) {
	api, _ := drawnFrom(t, sheets(4))
	alone(api)

	told := shaped(t, api)
	if told.GetPages() != 4 || len(told.GetSheets()) != 4 {
		t.Errorf("the document came back as %+v", told)
	}
}

// shaped is what a document is, as the window is told it.
func shaped(t *testing.T, api *API) *v1.GetDocumentResponse {
	t.Helper()
	out, err := api.GetDocument(t.Context(), connect.NewRequest(&v1.GetDocumentRequest{Path: book}))
	if err != nil {
		t.Fatalf("asked what the document is and was refused: %v", err)
	}
	return out.Msg
}

// A path that leaves the vault is refused, and so is one the vault holds
// nothing at. Neither is answered with anything drawn.
func TestAPathTheVaultDoesNotHoldIsRefused(t *testing.T) {
	for _, path := range []string{
		"../outside.pdf",
		"library/../../outside.pdf",
		"/etc/passwd",
		"library/nothing.pdf",
		beside,
	} {
		t.Run(path, func(t *testing.T) {
			from := sheets(4)
			api, handler := drawnFrom(t, from)
			alone(api)

			for _, url := range []string{
				assetOf(path),
				pageOf(path, 0, 400),
			} {
				out := ask(handler, url)
				if out.Code == http.StatusOK {
					t.Errorf("%s was answered", url)
				}
			}
			if opens, draws, _ := from.counted(); opens+draws != 0 {
				t.Errorf("a path the vault does not hold opened %d and drew %d", opens, draws)
			}
		})
	}
}

// A page the document does not have, and a width that is not one, are refused.
func TestAPageTheDocumentDoesNotHaveIsRefused(t *testing.T) {
	for _, one := range []struct {
		at   string
		wide string
		want int
	}{
		{"4", "?wide=400", http.StatusNotFound},
		{"99", "?wide=400", http.StatusNotFound},
		{"-1", "?wide=400", http.StatusBadRequest},
		{"one", "?wide=400", http.StatusBadRequest},
		{"0", "", http.StatusBadRequest},
		{"0", "?wide=0", http.StatusBadRequest},
		{"0", "?wide=99999", http.StatusBadRequest},
	} {
		asked := one.at + one.wide
		t.Run(asked, func(t *testing.T) {
			api, handler := drawnFrom(t, sheets(4))
			alone(api)

			out := ask(handler, assetOf(book)+"/"+pagesFacet+"/"+asked)
			if out.Code != one.want {
				t.Errorf("asking for %s was answered %d, not %d", asked, out.Code, one.want)
			}
		})
	}
}

// A page already drawn is not drawn again, and the document it came out of is
// not opened again for the next page of it.
func TestAPageDrawnIsNotDrawnAgain(t *testing.T) {
	from := sheets(4)
	api, handler := drawnFrom(t, from)
	alone(api)

	first := ask(handler, pageOf(book, 0, 400))
	if first.Code != http.StatusOK {
		t.Fatalf("asked for a page and got %d: %s", first.Code, first.Body)
	}
	opens, drawn, _ := from.counted()
	if opens != 1 {
		t.Fatalf("the document was opened %d times", opens)
	}

	again := ask(handler, pageOf(book, 0, 400))
	if again.Code != http.StatusOK {
		t.Fatalf("asked for the page again and got %d", again.Code)
	}
	if first.Body.String() != again.Body.String() {
		t.Error("the page came back drawn differently the second time")
	}
	if opensAgain, drawnAgain, _ := from.counted(); drawnAgain != drawn || opensAgain != opens {
		t.Errorf("the second ask opened %d and drew %d, having opened %d and drawn %d",
			opensAgain, drawnAgain, opens, drawn)
	}

	// Another page of the same document is drawn out of the document already
	// open.
	if out := ask(handler, pageOf(book, 2, 400)); out.Code != http.StatusOK {
		t.Fatalf("asked for another page and got %d", out.Code)
	}
	if opensAgain, _, _ := from.counted(); opensAgain != opens {
		t.Errorf("a second page opened the document %d times", opensAgain)
	}
}

// A page dropped from memory to stay inside the bound is drawn again when it is
// asked for, where nothing keeps it on disk.
func TestAPageDroppedForRoomIsDrawnAgain(t *testing.T) {
	from := sheets(4)
	api, handler := drawnFrom(t, from)
	alone(api)
	api.Viewer.drawn.Load().limit = 1

	if out := ask(handler, pageOf(book, 0, 400)); out.Code != http.StatusOK {
		t.Fatalf("asked for a page and got %d", out.Code)
	}
	_, drawn, _ := from.counted()

	if out := ask(handler, pageOf(book, 0, 400)); out.Code != http.StatusOK {
		t.Fatalf("asked for the page again and got %d", out.Code)
	}
	if _, drawnAgain, _ := from.counted(); drawnAgain <= drawn {
		t.Errorf("the page was not drawn again: %d drawings both times", drawn)
	}
}

// Many asks for one page at once draw it once, and every one of them is
// answered with that drawing.
func TestManyAsksForOnePageDrawItOnce(t *testing.T) {
	from := sheets(4)
	api, handler := drawnFrom(t, from)
	alone(api)

	const askers = 16
	bodies := make([]string, askers)
	codes := make([]int, askers)
	var asking sync.WaitGroup
	for i := range askers {
		asking.Add(1)
		go func() {
			defer asking.Done()
			out := ask(handler, pageOf(book, 0, 400))
			codes[i], bodies[i] = out.Code, out.Body.String()
		}()
	}
	asking.Wait()

	for i := range askers {
		if codes[i] != http.StatusOK {
			t.Fatalf("one of %d asks was answered %d", askers, codes[i])
		}
		if bodies[i] != bodies[0] {
			t.Errorf("two asks for one page came back drawn differently")
		}
	}
	opens, drawn, _ := from.counted()
	if opens != 1 {
		t.Errorf("%d asks opened the document %d times", askers, opens)
	}
	if drawn != 1 {
		t.Errorf("%d asks for one page drew it %d times", askers, drawn)
	}
}

// Many asks for a whole document at once open it once.
func TestManyAsksForOneDocumentOpenItOnce(t *testing.T) {
	from := sheets(4)
	api, _ := drawnFrom(t, from)
	alone(api)

	var asking sync.WaitGroup
	for range 16 {
		asking.Add(1)
		go func() {
			defer asking.Done()
			_, err := api.GetDocument(t.Context(), connect.NewRequest(&v1.GetDocumentRequest{Path: book}))
			if err != nil {
				t.Errorf("asked what the document is and was refused: %v", err)
			}
		}()
	}
	asking.Wait()

	if opens, _, _ := from.counted(); opens != 1 {
		t.Errorf("the document was opened %d times", opens)
	}
}

// The page after the one asked for is drawn before anybody asks for it.
func TestTheNextPageIsDrawnBeforeItIsAsked(t *testing.T) {
	from := sheets(4)
	_, handler := drawnFrom(t, from)

	if out := ask(handler, pageOf(book, 0, 400)); out.Code != http.StatusOK {
		t.Fatalf("asked for a page and got %d", out.Code)
	}
	eventually(t, "the next page was not drawn ahead", func() bool {
		return from.drewPage(1) == 1
	})

	if out := ask(handler, pageOf(book, 1, 400)); out.Code != http.StatusOK {
		t.Fatalf("asked for the next page and got %d", out.Code)
	}
	// Asking for it draws the page after it, and not it again.
	if drawn := from.drewPage(1); drawn != 1 {
		t.Errorf("the page drawn ahead was drawn %d times", drawn)
	}
}

// A document that cannot be reached inside the bound is answered as busy. The
// library waits minutes for a worker, and minutes is not an answer to somebody
// turning a page.
func TestADocumentThatCannotBeReachedInTimeIsBusy(t *testing.T) {
	from := sheets(4)
	from.gate = make(chan struct{})
	api, handler := drawnFrom(t, from)
	alone(api)
	api.Viewer.patience = 20 * time.Millisecond
	letIn := sync.OnceFunc(func() { close(from.gate) })
	t.Cleanup(letIn)

	// What a document is is asked of the schema and a page of a URL, and both
	// say the document is busy rather than holding the caller.
	_, err := api.GetDocument(t.Context(), connect.NewRequest(&v1.GetDocumentRequest{Path: book}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("what the document is was refused %v, not that it is busy", err)
	}
	out := ask(handler, pageOf(book, 0, 400))
	if out.Code != http.StatusServiceUnavailable {
		t.Errorf("a page was answered %d, not that the document is busy", out.Code)
	}
	if out.Header().Get("Retry-After") == "" {
		t.Error("a page was not told when to ask again")
	}

	// The opening went on, so the ask after it finds the document open.
	letIn()
	api.Viewer.patience = patience
	eventually(t, "the document never opened", func() bool {
		_, err := api.GetDocument(t.Context(), connect.NewRequest(&v1.GetDocumentRequest{Path: book}))
		return err == nil
	})
	if opens, _, _ := from.counted(); opens != 1 {
		t.Errorf("the document was opened %d times", opens)
	}
}

// A document nobody has looked at for a while is closed, and the worker it was
// holding goes back. Asking for it again opens it again.
func TestADocumentNobodyIsLookingAtIsClosed(t *testing.T) {
	from := sheets(4)
	api, handler := drawnFrom(t, from)
	alone(api)
	api.Viewer.docs.Load().idleFor = 10 * time.Millisecond

	if out := ask(handler, pageOf(book, 0, 400)); out.Code != http.StatusOK {
		t.Fatalf("asked for a page and got %d", out.Code)
	}
	eventually(t, "the document was never closed", func() bool {
		_, _, closed := from.counted()
		return closed == 1
	})

	if out := ask(handler, pageOf(book, 1, 400)); out.Code != http.StatusOK {
		t.Fatalf("asked for another page and got %d", out.Code)
	}
	if opens, _, _ := from.counted(); opens != 2 {
		t.Errorf("the document was opened %d times, having been closed", opens)
	}
}

// Only so many documents are held open at once. Each holds a worker of the
// pool, and a recognition holds another for as long as it reads.
func TestOnlySoManyDocumentsAreHeldOpen(t *testing.T) {
	from := sheets(4)
	api, handler := drawnFrom(t, from)
	alone(api)
	api.Viewer.docs.Load().limit = 1

	for _, path := range []string{book, another} {
		if out := ask(handler, pageOf(path, 0, 400)); out.Code != http.StatusOK {
			t.Fatalf("asked for a page of %s and got %d", path, out.Code)
		}
	}
	opens, _, closed := from.counted()
	if opens != 2 || closed != 1 {
		t.Errorf("two documents opened %d and closed %d, holding one at a time", opens, closed)
	}
}

// A page of a document the library draws comes back a picture of it.
func TestAPageOfARealDocumentComesBack(t *testing.T) {
	raw, err := os.ReadFile("../../internal/adapter/pdf/testdata/scan.pdf")
	if err != nil {
		t.Fatal(err)
	}
	api, handler := fromTheLibrary(t, string(raw))

	if told := shaped(t, api); told.GetPages() < 1 {
		t.Fatalf("the document came back as %+v", told)
	}

	page := ask(handler, pageOf(book, 0, 500))
	if page.Code != http.StatusOK {
		t.Fatalf("asked for a page and got %d: %s", page.Code, page.Body)
	}
	drawn, err := jpeg.Decode(page.Body)
	if err != nil {
		t.Fatalf("what came back is not a picture: %v", err)
	}
	if drawn.Bounds().Dx() != 500 {
		t.Errorf("asked for 500 pixels wide and got %d", drawn.Bounds().Dx())
	}
}

// A file that is not a document is refused, and says so.
func TestAFileThatIsNotADocumentIsRefused(t *testing.T) {
	api, _ := fromTheLibrary(t, "not a PDF at all")

	_, err := api.GetDocument(t.Context(), connect.NewRequest(&v1.GetDocumentRequest{Path: book}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("a file that is not a document was refused %v", err)
	}
}

// What the window may load is said on everything it is served, and a page of a
// document is not among the things it may make up for itself.
func TestTheWindowIsToldWhatItMayLoad(t *testing.T) {
	api, handler := drawnFrom(t, sheets(4))
	alone(api)

	allowed := ask(handler, pageOf(book, 0, 400)).Header().Get("Content-Security-Policy")
	if allowed == "" {
		t.Fatal("nothing was said about what the window may load")
	}
	for _, made := range []string{"data:", "blob:"} {
		if strings.Contains(allowed, made) {
			t.Errorf("the window may load %s: %s", made, allowed)
		}
	}
	if !strings.Contains(allowed, "img-src 'self'") {
		t.Errorf("a picture may come from anywhere: %s", allowed)
	}
}

// pdf.Open is what the application draws with, and what the viewer holds by
// default.
func TestTheApplicationDrawsWithTheLibrary(t *testing.T) {
	raw, err := os.ReadFile("../../internal/adapter/pdf/testdata/labels.pdf")
	if err != nil {
		t.Fatal(err)
	}
	opened, err := looking(pdf.Documents{}).open(raw)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()

	scan, err := pdf.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	defer scan.Close()
	if opened.Pages() != scan.Pages() {
		t.Errorf("the viewer opened %d pages and the library %d", opened.Pages(), scan.Pages())
	}
}
