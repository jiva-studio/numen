package hosttest_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

// A proofreader reads the live version another plugin made, and puts its own made from it.
func TestHostTestTextsReadLiveVersionAndSaveOwn(t *testing.T) {
	ctx := t.Context()
	file := []byte("%PDF")
	ocr := hosttest.NewTexts("v1", "org.example.ocr")
	ocr.File("a.pdf", file)
	if err := host.WriteText(ctx, ocr.GetStore("org.example.ocr"), "k.jsonl", []domain.TextSegment{
		{Text: "the "}, {Text: "end", Selector: domain.Selector(`{"page":2}`)},
	}); err != nil {
		t.Fatal(err)
	}
	a, err := ocr.PutVersion(ctx, host.TextPut{Path: "a.pdf", Key: "k", Kind: "recognition", InputHash: domain.HashContent(file), File: "k.jsonl"})
	if err != nil {
		t.Fatal(err)
	}

	fixer := ocr.As("org.example.fixer")
	listing, err := fixer.ListVersions(ctx, "a.pdf")
	if err != nil || listing.Live != a.GetID() {
		t.Fatalf("live %q (%v), want %q", listing.Live, err, a.GetID())
	}
	text, err := host.ReadWholeText(ctx, fixer, listing.Versions[0])
	if err != nil || text.Text != "the end" || text.GetSelector(5) != `{"page":2}` {
		t.Fatalf("read %q (%v)", text.Text, err)
	}
	if err := host.WriteText(ctx, fixer.GetStore("org.example.fixer"), "b.jsonl", []domain.TextSegment{{Text: "the end"}}); err != nil {
		t.Fatal(err)
	}
	b, err := fixer.PutVersion(ctx, host.TextPut{Path: "a.pdf", Key: "b", Source: a.GetID(), Kind: "recognition", InputHash: a.TextHash, File: "b.jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	if listing, err := fixer.ListVersions(ctx, "a.pdf"); err != nil || listing.Live != b.GetID() {
		t.Fatalf("live %q, want the proofreading", listing.Live)
	}

	ocr.GetStore("org.example.fixer").File("b.jsonl", []byte(`{"text":"changed"}`+"\n"))
	if listing, err := fixer.ListVersions(ctx, "a.pdf"); err != nil || listing.Live != a.GetID() {
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

func TestHostTestTextsWithKeepEachVaultApart(t *testing.T) {
	ctx := t.Context()
	physics := hosttest.NewTexts("v1", "org.example.ocr")
	other := physics.With("v2")
	if err := host.WriteText(ctx, other.GetStore("org.example.ocr"), "k.jsonl", []domain.TextSegment{{Text: "x"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := other.PutVersion(ctx, host.TextPut{Path: "a.pdf", Key: "k", Kind: "recognition", File: "k.jsonl"}); err != nil {
		t.Fatal(err)
	}

	inOther, err := physics.With("v2").ListVersions(ctx, "a.pdf")
	if err != nil {
		t.Fatal(err)
	}
	inShown, err := physics.ListVersions(ctx, "a.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(inOther.Versions) != 1 || len(inShown.Versions) != 0 {
		t.Errorf("v2 holds %d versions and v1 %d, want 1 and 0", len(inOther.Versions), len(inShown.Versions))
	}
}

func putFiveSegments(t *testing.T, texts *hosttest.Texts) domain.TextVersion {
	t.Helper()
	file := []byte("%PDF")
	texts.File("a.pdf", file)
	segments := make([]domain.TextSegment, 5)
	for i := range segments {
		segments[i] = domain.TextSegment{Text: "s" + string(rune('0'+i)) + " "}
	}
	if err := host.WriteText(t.Context(), texts.GetStore("org.example.ocr"), "k.jsonl", segments); err != nil {
		t.Fatal(err)
	}
	version, err := texts.PutVersion(t.Context(), host.TextPut{Path: "a.pdf", Key: "k", Kind: "recognition", InputHash: domain.HashContent(file), File: "k.jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	return version
}

func getSegmentTexts(segments []domain.TextSegment) []string {
	out := make([]string, len(segments))
	for i, one := range segments {
		out[i] = one.Text
	}
	return out
}

func TestReadTextReturnsPagesOfThePageSizeWithTheCursorOfTheNextSegment(t *testing.T) {
	texts := hosttest.NewTexts("v1", "org.example.ocr")
	version := putFiveSegments(t, texts)
	texts.SetPageSize(2)

	for _, one := range []struct {
		cursor, next string
		want         []string
	}{
		{"", "2", []string{"s0 ", "s1 "}},
		{"2", "4", []string{"s2 ", "s3 "}},
		{"4", "", []string{"s4 "}},
		{"5", "", []string{}},
	} {
		page, err := texts.ReadText(t.Context(), "a.pdf", version.GetID(), one.cursor)
		if err != nil {
			t.Fatalf("ReadText at the cursor %q: %v", one.cursor, err)
		}
		if got := getSegmentTexts(page.Segments); !slices.Equal(got, one.want) || page.NextCursor != one.next {
			t.Errorf("ReadText at the cursor %q returned %q with the next cursor %q, want %q and %q", one.cursor, got, page.NextCursor, one.want, one.next)
		}
	}
}

func TestReadTextReturnsUpToAThousandSegmentsInOnePageByDefault(t *testing.T) {
	texts := hosttest.NewTexts("v1", "org.example.ocr")
	file := []byte("%PDF")
	texts.File("a.pdf", file)
	segments := make([]domain.TextSegment, 1001)
	for i := range segments {
		segments[i] = domain.TextSegment{Text: "w "}
	}
	if err := host.WriteText(t.Context(), texts.GetStore("org.example.ocr"), "k.jsonl", segments); err != nil {
		t.Fatal(err)
	}
	version, err := texts.PutVersion(t.Context(), host.TextPut{Path: "a.pdf", Key: "k", Kind: "recognition", InputHash: domain.HashContent(file), File: "k.jsonl"})
	if err != nil {
		t.Fatal(err)
	}

	first, err := texts.ReadText(t.Context(), "a.pdf", version.GetID(), "")
	if err != nil || len(first.Segments) != 1000 || first.NextCursor != "1000" {
		t.Fatalf("the first page has %d segments and the next cursor %q (%v), want 1000 and \"1000\"", len(first.Segments), first.NextCursor, err)
	}
	last, err := texts.ReadText(t.Context(), "a.pdf", version.GetID(), first.NextCursor)
	if err != nil || len(last.Segments) != 1 || last.NextCursor != "" {
		t.Errorf("the last page has %d segments and the next cursor %q (%v), want 1 and none", len(last.Segments), last.NextCursor, err)
	}
}

func TestReadTextAtACursorThatIsNotANumberOrPastTheEndIsNotExist(t *testing.T) {
	texts := hosttest.NewTexts("v1", "org.example.ocr")
	version := putFiveSegments(t, texts)
	texts.SetPageSize(2)

	for _, cursor := range []string{"two", "-1", "6"} {
		if _, err := texts.ReadText(t.Context(), "a.pdf", version.GetID(), cursor); !errors.Is(err, host.ErrNotExist) {
			t.Errorf("ReadText at the cursor %q returned %v, want host.ErrNotExist", cursor, err)
		}
	}
}
