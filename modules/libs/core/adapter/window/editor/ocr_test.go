package editor

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

// openHighlightWindow is a window over a vault holding one document with a text layer, and
// that document read, so a test can name a word and ask where it is.
func openHighlightWindow(t *testing.T) (*API, http.Handler, *pdf.Book) {
	t.Helper()
	raw, err := os.ReadFile("../../../internal/adapter/pdf/testdata/tiny.pdf")
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
func where(t *testing.T, doc *pdf.Book, word string) *v1.Span {
	t.Helper()
	at := strings.Index(doc.Text, word)
	if at < 0 {
		t.Fatalf("the document does not say %q", word)
	}
	return &v1.Span{From: int32(at), To: int32(at + len(word))}
}

// reads is what the runs of a source's text say and where they sit, as the
// window is told it.
func reads(api *API, path string, at ...*v1.Span) ([]*v1.Run, error) {
	out, err := api.ReadOcr(context.Background(), connect.NewRequest(&v1.ReadOcrRequest{
		Path: path, Spans: at,
	}))
	if err != nil {
		return nil, err
	}
	return out.Msg.GetRuns(), nil
}

// A run of a source's text comes back as what it says and the boxes covering
// it, each on the page it was read from.
func TestARunOfTheProseComesBackAsTextAndBoxes(t *testing.T) {
	api, _, doc := openHighlightWindow(t)

	runs, err := reads(api, book, where(t, doc, "Delta"))
	if err != nil {
		t.Fatalf("asked where a word is and was refused: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("one place was asked about and %d came back: %+v", len(runs), runs)
	}
	if runs[0].GetText() != "Delta" {
		t.Errorf("the run reads %q", runs[0].GetText())
	}
	boxes := runs[0].GetBoxes()
	if len(boxes) != 1 || boxes[0].GetPage() != 1 {
		t.Fatalf("%q is on the second page and came back at %+v", "Delta", boxes)
	}
	rect := boxes[0].GetRect()
	if rect.GetMinX() < 0 || rect.GetMinY() < 0 || rect.GetMaxX() > 1 || rect.GetMaxY() > 1 {
		t.Errorf("%q is at %+v, which is off the page", "Delta", rect)
	}
	if rect.GetMinX() >= rect.GetMaxX() || rect.GetMinY() >= rect.GetMaxY() {
		t.Errorf("%q is at %+v, which is nothing at all", "Delta", rect)
	}
	span := boxes[0].GetSpan()
	if span.GetFrom() >= span.GetTo() {
		t.Errorf("%q covers %+v, which is nothing of the text", "Delta", span)
	}
}

// A source the index does not hold says nothing and is lit nowhere. It keeps
// its place in the answer, so a caller still reads one run per run it asked
// about.
func TestASourceNothingIsKnownAboutComesBackEmpty(t *testing.T) {
	api, _, doc := openHighlightWindow(t)
	api.Highlight.Sources = indexed{}

	runs, err := reads(api, book, where(t, doc, "Delta"))
	if err != nil {
		t.Fatalf("asked where a word is and was refused: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("one place was asked about and %d came back: %+v", len(runs), runs)
	}
	if runs[0].GetText() != "" || len(runs[0].GetBoxes()) != 0 {
		t.Errorf("a source nothing is known about came back as %+v", runs[0])
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
			api, _, _ := openHighlightWindow(t)

			_, err := reads(api, path, &v1.Span{From: 0, To: 5})
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
		at   []*v1.Span
	}{
		{"no run at all", nil},
		{"a place before the text", []*v1.Span{{From: -1, To: 4}}},
		{"a run of nothing", []*v1.Span{{From: 0, To: 0}}},
		{"a run longer than a book", []*v1.Span{{From: 0, To: longestRun + 1}}},
	} {
		t.Run(one.what, func(t *testing.T) {
			api, _, _ := openHighlightWindow(t)

			if _, err := reads(api, book, one.at...); connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Errorf("asking about %s was refused %v", one.what, err)
			}
		})
	}
}
