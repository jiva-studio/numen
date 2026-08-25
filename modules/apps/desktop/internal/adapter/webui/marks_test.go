package webui

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/pdf"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/source"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
)

// indexed is what the index holds about the sources of the vault under test.
// Where a passage sits turns on one of its answers, and nothing here asks it
// the others.
type indexed map[string]port.Recognised

func (i indexed) Reading(_ context.Context, _, path string) (port.Recognised, bool, error) {
	found, held := i[path]
	return found, held, nil
}

func (i indexed) Fingerprints(context.Context, string, domain.SourceKind) (map[string]domain.FileRef, error) {
	return nil, nil
}

func (i indexed) Unchunked(context.Context, string, domain.SourceKind, int) ([]string, error) {
	return nil, nil
}

func (i indexed) ByOtherRecipe(context.Context, string, domain.SourceKind, []string, int) ([]string, error) {
	return nil, nil
}

func (i indexed) Recognised(context.Context, string, domain.SourceKind) ([]port.Recognised, error) {
	return nil, nil
}

// placing is a window over a vault holding one document with a text layer, and
// that document read, so a test can name a word and ask where it is.
func placing(t *testing.T) (*API, http.Handler, *pdf.Book) {
	t.Helper()
	raw, err := os.ReadFile("../pdf/testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := pdf.Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	vault := testsupport.NewVault(t, map[string]string{
		book:   string(raw),
		beside: "a file the vault leaves alone",
	})
	api := &API{
		Readers: filesystem.Readers{},
		Marking: &source.Marks{
			Readers:   filesystem.Readers{},
			Sources:   indexed{book: {Path: book}},
			Documents: pdf.Documents{},
		},
	}
	api.show(vault)
	return api, api.Serving(http.NotFoundHandler()), doc
}

// where is where a word of the document is, as the window would ask about it.
func where(t *testing.T, doc *pdf.Book, word string) string {
	t.Helper()
	at := strings.Index(doc.Text, word)
	if at < 0 {
		t.Fatalf("the document does not say %q", word)
	}
	return marksOf(book, at, len(word))
}

// A run of a source's text comes back as the pages it falls on and, on each,
// the rectangles covering it.
func TestARunOfTheProseComesBackAsPagesAndRectangles(t *testing.T) {
	_, handler, doc := placing(t)

	out := ask(handler, where(t, doc, "Delta"))
	if out.Code != http.StatusOK {
		t.Fatalf("asked where a word is and got %d: %s", out.Code, out.Body)
	}
	if said := out.Header().Get("Content-Type"); said != "application/json" {
		t.Errorf("where the word is came back as %q", said)
	}
	if said := out.Header().Get("Cache-Control"); said != "no-store" {
		t.Errorf("where the word is is kept: %q", said)
	}

	var told covering
	if err := json.NewDecoder(out.Body).Decode(&told); err != nil {
		t.Fatal(err)
	}
	if len(told.Runs) != 1 {
		t.Fatalf("one place was asked about and %d came back: %+v", len(told.Runs), told.Runs)
	}
	marks := told.Runs[0].Marks
	if len(marks) != 1 || marks[0].Page != 1 || len(marks[0].Rects) != 1 {
		t.Fatalf("%q is on the second page and came back at %+v", "Delta", marks)
	}
	box := marks[0].Rects[0]
	if box.MinX < 0 || box.MinY < 0 || box.MaxX > 1 || box.MaxY > 1 {
		t.Errorf("%q is at %+v, which is off the page", "Delta", box)
	}
	if box.MinX >= box.MaxX || box.MinY >= box.MaxY {
		t.Errorf("%q is at %+v, which is nothing at all", "Delta", box)
	}
}

// A source the index does not hold is lit nowhere, and the window is told a
// list of no pages.
func TestASourceNothingIsKnownAboutComesBackWithNoPages(t *testing.T) {
	api, handler, doc := placing(t)
	api.Marking.Sources = indexed{}

	out := ask(handler, where(t, doc, "Delta"))
	if out.Code != http.StatusOK {
		t.Fatalf("asked where a word is and got %d: %s", out.Code, out.Body)
	}
	if body := strings.TrimSpace(out.Body.String()); body != `{"runs":[]}` {
		t.Errorf("a source nothing is known about came back as %s", body)
	}
}

// A path the vault does not hold is refused, and so is a path that leaves it
// and a file the vault holds as nothing. The path goes to the vault the way
// every path from outside does, and it answers for all three.
func TestAPathTheVaultDoesNotHoldIsNotLit(t *testing.T) {
	for _, path := range []string{
		"../outside.pdf",
		"library/../../outside.pdf",
		"library/nothing.pdf",
		beside,
	} {
		t.Run(path, func(t *testing.T) {
			_, handler, _ := placing(t)

			out := ask(handler, marksOf(path, 0, 5))
			if out.Code != http.StatusNotFound {
				t.Errorf("%s was answered %d, not %d", path, out.Code, http.StatusNotFound)
			}
		})
	}
}

// A run that is not one is refused, and nothing is read to say so.
func TestARunThatIsNotOneIsRefused(t *testing.T) {
	for _, query := range []string{
		"start=0",
		"length=5",
		"start=-1&length=5",
		"start=nowhere&length=5",
		"start=0&length=0",
		"start=0&length=none",
	} {
		t.Run(query, func(t *testing.T) {
			_, handler, _ := placing(t)

			out := ask(handler, assetOf(book)+"/"+marksFacet+"?"+query)
			if out.Code != http.StatusBadRequest {
				t.Errorf("asking for %s was answered %d", query, out.Code)
			}
		})
	}
}

// A build with nothing to place a passage with says so, and the rest of the
// window works as it did.
func TestABuildThatCannotPlaceAPassageSaysSo(t *testing.T) {
	api, handler, doc := placing(t)
	api.Marking = nil

	out := ask(handler, where(t, doc, "Delta"))
	if out.Code != http.StatusNotImplemented {
		t.Errorf("a build with nothing to place a passage with answered %d", out.Code)
	}
}
