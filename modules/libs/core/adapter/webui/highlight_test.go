package webui

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/pdf"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// indexed is what the index holds about the sources of the vault under test.
// Where a passage sits turns on one of its answers, and nothing here asks it
// the others.
type indexed map[string]port.SourceText

func (i indexed) Reading(_ context.Context, _ domain.VaultID, path string) (port.SourceText, bool, error) {
	found, held := i[path]
	return found, held, nil
}

func (i indexed) Fingerprints(context.Context, domain.VaultID, domain.SourceKind) (map[string]domain.Fingerprint, error) {
	return nil, nil
}

func (i indexed) Unchunked(context.Context, domain.VaultID, domain.SourceKind, int) ([]string, error) {
	return nil, nil
}

func (i indexed) ByOtherRecipe(context.Context, domain.VaultID, domain.SourceKind, []string, int) ([]string, error) {
	return nil, nil
}

func (i indexed) Recognised(context.Context, domain.VaultID, domain.SourceKind) ([]port.SourceText, error) {
	return nil, nil
}

func (i indexed) Under(context.Context, domain.VaultID, string) ([]domain.Fingerprint, error) {
	return nil, nil
}

// placing is a window over a vault holding one document with a text layer, and
// that document read, so a test can name a word and ask where it is.
func placing(t *testing.T) (*API, http.Handler, *pdf.Book) {
	t.Helper()
	raw, err := os.ReadFile("../../internal/adapter/pdf/testdata/tiny.pdf")
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
		Readers: filesystem.VaultReaders{},
		Highlight: &source.Highlight{
			Readers:   filesystem.VaultReaders{},
			Sources:   indexed{book: {Fingerprint: domain.Fingerprint{Path: book}}},
			Documents: pdf.Documents{},
		},
	}
	api.show(vault)
	return api, api.Serving(http.NotFoundHandler()), doc
}

// where is where a word of the document is, as the window would ask about it.
func where(t *testing.T, doc *pdf.Book, word string) *v1.Stretch {
	t.Helper()
	at := strings.Index(doc.Text, word)
	if at < 0 {
		t.Fatalf("the document does not say %q", word)
	}
	return &v1.Stretch{Start: int32(at), Length: int32(len(word))}
}

// highlights is where the runs of a source's text sit, as the window is told it.
func highlights(api *API, path string, at ...*v1.Stretch) ([]*v1.Highlight, error) {
	out, err := api.ListHighlights(context.Background(), connect.NewRequest(&v1.ListHighlightsRequest{
		Path: path, At: at,
	}))
	if err != nil {
		return nil, err
	}
	return out.Msg.GetRuns(), nil
}

// A run of a source's text comes back as the pages it falls on and, on each,
// the rectangles covering it.
func TestARunOfTheProseComesBackAsPagesAndRectangles(t *testing.T) {
	api, _, doc := placing(t)

	runs, err := highlights(api, book, where(t, doc, "Delta"))
	if err != nil {
		t.Fatalf("asked where a word is and was refused: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("one place was asked about and %d came back: %+v", len(runs), runs)
	}
	pages := runs[0].GetPages()
	if len(pages) != 1 || pages[0].GetIndex() != 1 || len(pages[0].GetRects()) != 1 {
		t.Fatalf("%q is on the second page and came back at %+v", "Delta", pages)
	}
	box := pages[0].GetRects()[0]
	if box.GetMinX() < 0 || box.GetMinY() < 0 || box.GetMaxX() > 1 || box.GetMaxY() > 1 {
		t.Errorf("%q is at %+v, which is off the page", "Delta", box)
	}
	if box.GetMinX() >= box.GetMaxX() || box.GetMinY() >= box.GetMaxY() {
		t.Errorf("%q is at %+v, which is nothing at all", "Delta", box)
	}
}

// A source the index does not hold is lit nowhere, and the window is told a
// list of no pages.
func TestASourceNothingIsKnownAboutComesBackWithNoPages(t *testing.T) {
	api, _, doc := placing(t)
	api.Highlight.Sources = indexed{}

	runs, err := highlights(api, book, where(t, doc, "Delta"))
	if err != nil {
		t.Fatalf("asked where a word is and was refused: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("a source nothing is known about came back as %+v", runs)
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
			api, _, _ := placing(t)

			_, err := highlights(api, path, &v1.Stretch{Start: 0, Length: 5})
			if code := connect.CodeOf(err); code != connect.CodeNotFound &&
				code != connect.CodeInvalidArgument {
				t.Errorf("%s was refused %v", path, err)
			}
		})
	}
}

// A run that is not one is refused, and nothing is read to say so.
func TestARunThatIsNotOneIsRefused(t *testing.T) {
	for _, one := range []struct {
		what string
		at   []*v1.Stretch
	}{
		{"no run at all", nil},
		{"a place before the text", []*v1.Stretch{{Start: -1, Length: 5}}},
		{"a run of nothing", []*v1.Stretch{{Start: 0, Length: 0}}},
		{"a run longer than a book", []*v1.Stretch{{Start: 0, Length: longestRun + 1}}},
	} {
		t.Run(one.what, func(t *testing.T) {
			api, _, _ := placing(t)

			if _, err := highlights(api, book, one.at...); connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Errorf("asking about %s was refused %v", one.what, err)
			}
		})
	}
}
