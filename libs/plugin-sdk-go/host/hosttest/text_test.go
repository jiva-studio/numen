package hosttest_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
)

// A proofreader reads the live version another plugin made, and puts its own made from it.
func TestHostTestTextsReadLiveVersionAndSaveOwn(t *testing.T) {
	ctx := t.Context()
	file := []byte("%PDF")
	ocr := hosttest.NewTexts("v1", "org.example.ocr")
	ocr.File("a.pdf", file)
	if err := host.WriteText(ctx, ocr.GetStore("org.example.ocr"), "k.jsonl", []domain.TextSegment{
		{Text: "teh "}, {Text: "end", Selector: host.Selector{Page: 2}.Encode()},
	}); err != nil {
		t.Fatal(err)
	}
	a, err := ocr.PutVersion(ctx, host.TextPut{Path: "a.pdf", Key: "k", Kind: "recognition", SourceHash: domain.HashContent(file), File: "k.jsonl"})
	if err != nil {
		t.Fatal(err)
	}

	fixer := ocr.As("org.example.fixer")
	listing, err := fixer.ListVersions(ctx, "a.pdf")
	if err != nil || listing.Live != a.GetID() {
		t.Fatalf("live %q (%v), want %q", listing.Live, err, a.GetID())
	}
	text, err := host.ReadWholeText(ctx, fixer, listing.Versions[0])
	if err != nil || text.Text != "teh end" || text.GetSelector(5) != `{"page":2}` {
		t.Fatalf("read %q (%v)", text.Text, err)
	}
	if err := host.WriteText(ctx, fixer.GetStore("org.example.fixer"), "b.jsonl", []domain.TextSegment{{Text: "the end"}}); err != nil {
		t.Fatal(err)
	}
	b, err := fixer.PutVersion(ctx, host.TextPut{Path: "a.pdf", Key: "b", Source: a.GetID(), Kind: "recognition", SourceHash: a.Hash, File: "b.jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	if listing, _ := fixer.ListVersions(ctx, "a.pdf"); listing.Live != b.GetID() {
		t.Fatalf("live %q, want the proofreading", listing.Live)
	}

	ocr.GetStore("org.example.fixer").(*hosttest.Store).File("b.jsonl", []byte(`{"text":"changed"}`+"\n"))
	if listing, _ := fixer.ListVersions(ctx, "a.pdf"); listing.Live != a.GetID() {
		t.Fatalf("live %q with the proofreading's file changed, want the recognition", listing.Live)
	}
	if got := ocr.TakeChanges(); !slices.Equal(got, []string{"a.pdf", "a.pdf"}) {
		t.Fatalf("changes %v", got)
	}
}

func TestAVersionThatIsNotOneIsRefused(t *testing.T) {
	texts := hosttest.NewTexts("v1", "org.example.ocr")
	if err := host.WriteText(t.Context(), texts.GetStore("org.example.ocr"), "k.jsonl", []domain.TextSegment{{Text: "a"}}); err != nil {
		t.Fatal(err)
	}
	for name, params := range map[string]host.TextPut{
		"no key":             {Path: "a.pdf", Kind: "recognition", File: "k.jsonl"},
		"no kind":            {Path: "a.pdf", Key: "k", File: "k.jsonl"},
		"no file":            {Path: "a.pdf", Key: "k", Kind: "recognition"},
		"a file not there":   {Path: "a.pdf", Key: "k", Kind: "recognition", File: "gone.jsonl"},
		"a source not there": {Path: "a.pdf", Key: "k", Kind: "recognition", File: "k.jsonl", Source: "x/y"},
	} {
		if _, err := texts.PutVersion(t.Context(), params); !errors.Is(err, host.ErrInvalidVersion) {
			t.Errorf("%s: got %v", name, err)
		}
	}
	if _, err := texts.ReadText(t.Context(), "a.pdf", "x/y", ""); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("a version not there: got %v", err)
	}
}
