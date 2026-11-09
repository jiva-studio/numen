package domain_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func getEpoch() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

func isAllIntact(domain.TextVersion) bool { return true }

func TestFindLiveVersionReturnsTheNewestValidVersion(t *testing.T) {
	versions := []domain.TextVersion{
		{Path: "a.pdf", PluginID: "fixer", Key: "b", Source: "reader/a", InputHash: "ha", TextHash: "hb", CreateTime: getEpoch().Add(time.Minute)},
		{Path: "a.pdf", PluginID: "reader", Key: "a", InputHash: "file-1", TextHash: "ha", CreateTime: getEpoch()},
		{Path: "a.pdf", PluginID: "stale", Key: "c", InputHash: "old-file", TextHash: "hc", CreateTime: getEpoch().Add(2 * time.Minute)},
	}
	got, ok := domain.FindLiveVersion(versions, "file-1", nil, isAllIntact)
	if !ok || got.GetID() != "fixer/b" {
		t.Fatalf("got %q (%v), want fixer/b", got.GetID(), ok)
	}
}

func TestFindLiveVersionBreaksATieByListOrder(t *testing.T) {
	versions := []domain.TextVersion{
		{PluginID: "p", Key: "first", InputHash: "f", CreateTime: getEpoch()},
		{PluginID: "p", Key: "second", InputHash: "f", CreateTime: getEpoch()},
	}
	got, _ := domain.FindLiveVersion(versions, "f", nil, isAllIntact)
	if got.Key != "second" {
		t.Fatalf("got %q, want the later entry", got.Key)
	}
}

func TestFindLiveVersionSkipsAVersionWhoseFileChanged(t *testing.T) {
	versions := []domain.TextVersion{
		{PluginID: "reader", Key: "a", InputHash: "f", TextHash: "ha", CreateTime: getEpoch()},
		{PluginID: "fixer", Key: "b", Source: "reader/a", InputHash: "ha", TextHash: "hb", CreateTime: getEpoch().Add(time.Minute)},
	}
	got, ok := domain.FindLiveVersion(versions, "f", nil, func(v domain.TextVersion) bool { return v.PluginID != "fixer" })
	if !ok || got.GetID() != "reader/a" {
		t.Fatalf("got %q (%v), want the version before the changed one", got.GetID(), ok)
	}
}

func TestIsChainValidFailsWhenTheSourceChanged(t *testing.T) {
	versions := []domain.TextVersion{
		{PluginID: "reader", Key: "a", InputHash: "file-1", TextHash: "ha-2"},
		{PluginID: "fixer", Key: "b", Source: "reader/a", InputHash: "ha-1", TextHash: "hb"},
	}
	if !domain.IsChainValid(versions, "reader/a", "file-1", nil) {
		t.Fatal("the version from the file is valid")
	}
	if domain.IsChainValid(versions, "fixer/b", "file-1", nil) {
		t.Fatal("a version made from other text of its source is valid")
	}
}

func TestIsChainValidFailsWhenTheFileChanged(t *testing.T) {
	versions := []domain.TextVersion{{PluginID: "reader", Key: "a", InputHash: "file-1"}}
	if domain.IsChainValid(versions, "reader/a", "file-2", nil) {
		t.Fatal("a version of other bytes of the file is valid")
	}
}

func TestIsChainValidFailsOnAMissingSourceAndOnACycle(t *testing.T) {
	if domain.IsChainValid(nil, "missing/id", "", nil) {
		t.Fatal("an unknown version id is valid")
	}
	missing := []domain.TextVersion{{PluginID: "fixer", Key: "b", Source: "reader/a"}}
	if domain.IsChainValid(missing, "fixer/b", "f", nil) {
		t.Fatal("a version whose source is gone is valid")
	}
	cycle := []domain.TextVersion{
		{PluginID: "p", Key: "x", Source: "p/y", InputHash: "hy", TextHash: "hx"},
		{PluginID: "p", Key: "y", Source: "p/x", InputHash: "hx", TextHash: "hy"},
	}
	if domain.IsChainValid(cycle, "p/x", "f", nil) {
		t.Fatal("a version reaching itself is valid")
	}
}

func getPropertyVersion() domain.TextVersion {
	return domain.TextVersion{
		Path: "link.md", PluginID: "importer", Key: "a", TextHash: "ha",
		SourceProperty: domain.TextSourceProperty{Key: "url", Value: "https://example.com/talk"},
	}
}

func TestIsChainValidAcceptsAPropertyVersionWhileThePropertyIsUnchanged(t *testing.T) {
	versions := []domain.TextVersion{getPropertyVersion()}
	properties := map[string]any{"url": "https://example.com/talk", "title": "Edited title"}
	if !domain.IsChainValid(versions, "importer/a", "any-file-hash", properties) {
		t.Fatal("a version made from a property that still holds its value is not valid")
	}
}

func TestIsChainValidFailsAPropertyVersionWhenThePropertyChanged(t *testing.T) {
	versions := []domain.TextVersion{getPropertyVersion()}
	cases := map[string]map[string]any{
		"another value": {"url": "https://example.com/other"},
		"missing key":   {"title": "Talk"},
		"no properties": nil,
		"not a string":  {"url": []any{"https://example.com/talk"}},
	}
	for name, properties := range cases {
		if domain.IsChainValid(versions, "importer/a", "f", properties) {
			t.Fatalf("%s: a version made from a property that no longer holds its value is valid", name)
		}
	}
}

func TestIsChainValidChainsAVersionDerivedFromAPropertyVersionByHash(t *testing.T) {
	versions := []domain.TextVersion{
		getPropertyVersion(),
		{PluginID: "fixer", Key: "b", Source: "importer/a", InputHash: "ha", TextHash: "hb"},
	}
	if !domain.IsChainValid(versions, "fixer/b", "f", map[string]any{"url": "https://example.com/talk"}) {
		t.Fatal("a version derived from a valid property version is not valid")
	}
	if domain.IsChainValid(versions, "fixer/b", "f", map[string]any{"url": "https://example.com/other"}) {
		t.Fatal("a version derived from a property version whose property changed is valid")
	}
}

func TestFindLiveVersionKeepsAPropertyVersionLiveAfterAnEditOfTheNote(t *testing.T) {
	versions := []domain.TextVersion{getPropertyVersion()}
	got, ok := domain.FindLiveVersion(versions, "edited-file", map[string]any{"url": "https://example.com/talk"}, isAllIntact)
	if !ok || got.GetID() != "importer/a" {
		t.Fatalf("got %q (%v), want importer/a", got.GetID(), ok)
	}
	if _, ok := domain.FindLiveVersion(versions, "edited-file", map[string]any{"url": "https://example.com/new"}, isAllIntact); ok {
		t.Fatal("a version made from a property that changed is live")
	}
}

func TestIsFromFileIsFalseForAPropertyVersion(t *testing.T) {
	if getPropertyVersion().IsFromFile() {
		t.Fatal("a version made from a property is made from the file")
	}
}

func TestFileInputComparesExtensionsWithoutCase(t *testing.T) {
	in := domain.FileInput{".pdf", ".pdf/readme"}
	if !in.IsFileAccepted("books/A.PDF") || !in.IsFileAccepted(".pdf") {
		t.Fatal("A.PDF and .pdf should be accepted")
	}
	if in.IsFileAccepted("books.pdf/readme") || in.IsFileAccepted("notes/a.md") || in.IsFileAccepted("readme") {
		t.Fatal("a file of another type is accepted")
	}
}

func TestVersionTextLocatesHeadingsAndSelectors(t *testing.T) {
	text := domain.NewVersionText(domain.TextVersion{Kind: "recognition"}, []domain.TextSegment{
		{Text: "Intro ", Heading: "One", Selector: `{"page":1}`},
		{Text: "body ", Selector: `{"page":2}`},
		{Text: "end", Heading: "Two"},
	})
	if text.Text != "Intro body end" {
		t.Fatalf("text %q", text.Text)
	}
	if got := text.Locate(-1); got != "" {
		t.Fatalf("locate -1: %q", got)
	}
	if got := text.Locate(6); got != "One" {
		t.Fatalf("locate 6: %q", got)
	}
	if got := text.Locate(7); got != "One" {
		t.Fatalf("locate 7: %q", got)
	}
	if got := text.Locate(11); got != "Two" {
		t.Fatalf("locate 11: %q", got)
	}
	if got := text.GetSelector(-1); got != "" {
		t.Fatalf("selector at -1: %q", got)
	}
	if got := text.GetSelector(0); got != `{"page":1}` {
		t.Fatalf("selector at 0: %q", got)
	}
	if got := text.GetSelector(6); got != `{"page":2}` {
		t.Fatalf("selector at 6: %q", got)
	}
	if got := text.GetSelector(7); got != `{"page":2}` {
		t.Fatalf("selector at 7: %q", got)
	}
	if got := text.GetSelector(12); got != "" {
		t.Fatalf("selector of a segment without one: %q", got)
	}
	if got := text.GetHeadings(); len(got) != 2 || got[1].Offset != 11 || got[1].Title != "Two" {
		t.Fatalf("headings %+v", got)
	}
	if got := text.GetHeadingsAt(6); len(got) != 0 {
		t.Fatalf("headings at 6: %v", got)
	}
	if got := text.GetHeadingsAt(11); len(got) != 1 || got[0] != "Two" {
		t.Fatalf("headings at 11: %v", got)
	}
	if got := len(text.GetSegments()); got != 3 {
		t.Fatalf("%d segments", got)
	}
}

func TestGetSegmentsInTrimsSegmentsToRangeWithHeadingAndSelector(t *testing.T) {
	text := domain.NewVersionText(domain.TextVersion{Kind: "recognition"}, []domain.TextSegment{
		{Text: "abc", Heading: "One", Selector: "s1"},
		{Text: "", Heading: "Empty"},
		{Text: "defg", Selector: "s2"},
		{Text: "hi", Heading: "Three", Selector: "s3"},
	})
	for name, tc := range map[string]struct {
		from, to int
		want     []domain.TextSegment
	}{
		"inside one":  {1, 2, []domain.TextSegment{{Text: "b", Heading: "One", Selector: "s1"}}},
		"across two":  {2, 5, []domain.TextSegment{{Text: "c", Heading: "One", Selector: "s1"}, {Text: "de", Selector: "s2"}}},
		"one whole":   {3, 7, []domain.TextSegment{{Text: "defg", Selector: "s2"}}},
		"to the end":  {7, 9, []domain.TextSegment{{Text: "hi", Heading: "Three", Selector: "s3"}}},
		"empty range": {4, 4, nil},
	} {
		got := text.GetSegmentsIn(tc.from, tc.to)
		if len(got) != len(tc.want) {
			t.Errorf("%s: got %+v, want %+v", name, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%s: got %+v, want %+v", name, got, tc.want)
			}
		}
	}
}
