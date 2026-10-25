package domain_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

var epoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func allEnabled(string) bool { return true }

func TestFindLiveVersionTakesTheNewestLiveVersion(t *testing.T) {
	versions := []domain.TextVersion{
		{Path: "a.pdf", PluginID: "reader", Key: "a", InputHash: "file-1", Hash: "ha", CreateTime: epoch},
		{Path: "a.pdf", PluginID: "fixer", Key: "b", Input: "reader/a", InputHash: "ha", Hash: "hb", CreateTime: epoch.Add(time.Minute)},
	}
	got, ok := domain.FindLiveVersion(versions, "file-1", allEnabled)
	if !ok || got.GetID() != "fixer/b" {
		t.Fatalf("got %q (%v), want fixer/b", got.GetID(), ok)
	}
}

func TestFindLiveVersionBreaksATieByListOrder(t *testing.T) {
	versions := []domain.TextVersion{
		{PluginID: "p", Key: "first", InputHash: "f", CreateTime: epoch},
		{PluginID: "p", Key: "second", InputHash: "f", CreateTime: epoch},
	}
	got, _ := domain.FindLiveVersion(versions, "f", allEnabled)
	if got.Key != "second" {
		t.Fatalf("got %q, want the later entry", got.Key)
	}
}

func TestIsLiveVersionFailsWhenTheInputChanged(t *testing.T) {
	versions := []domain.TextVersion{
		{PluginID: "reader", Key: "a", InputHash: "file-1", Hash: "ha-2"},
		{PluginID: "fixer", Key: "b", Input: "reader/a", InputHash: "ha-1", Hash: "hb"},
	}
	if !domain.IsLiveVersion(versions, "reader/a", "file-1", allEnabled) {
		t.Fatal("the version from the file is live")
	}
	if domain.IsLiveVersion(versions, "fixer/b", "file-1", allEnabled) {
		t.Fatal("a version made from other text of its input is live")
	}
}

func TestIsLiveVersionFailsWhenTheFileChanged(t *testing.T) {
	versions := []domain.TextVersion{{PluginID: "reader", Key: "a", InputHash: "file-1"}}
	if domain.IsLiveVersion(versions, "reader/a", "file-2", allEnabled) {
		t.Fatal("a version of other bytes of the file is live")
	}
}

func TestIsLiveVersionFailsOnADisabledPluginInTheChain(t *testing.T) {
	versions := []domain.TextVersion{
		{PluginID: "ocr", Key: "c", InputHash: "f", Hash: "hc"},
		{PluginID: "fixer", Key: "d", Input: "ocr/c", InputHash: "hc"},
	}
	isEnabled := func(id string) bool { return id != "ocr" }
	if domain.IsLiveVersion(versions, "fixer/d", "f", isEnabled) {
		t.Fatal("a version made from a disabled plugin's version is live")
	}
}

func TestIsLiveVersionFailsOnAMissingInputAndOnACycle(t *testing.T) {
	missing := []domain.TextVersion{{PluginID: "fixer", Key: "b", Input: "reader/a"}}
	if domain.IsLiveVersion(missing, "fixer/b", "f", allEnabled) {
		t.Fatal("a version whose input is gone is live")
	}
	cycle := []domain.TextVersion{
		{PluginID: "p", Key: "x", Input: "p/y", InputHash: "hy", Hash: "hx"},
		{PluginID: "p", Key: "y", Input: "p/x", InputHash: "hx", Hash: "hy"},
	}
	if domain.IsLiveVersion(cycle, "p/x", "f", allEnabled) {
		t.Fatal("a version reaching itself is live")
	}
}

func TestHasPluginInChain(t *testing.T) {
	versions := []domain.TextVersion{
		{PluginID: "ocr", Key: "c"},
		{PluginID: "fixer", Key: "d", Input: "ocr/c"},
	}
	if !domain.HasPluginInChain(versions, "fixer/d", "ocr") {
		t.Fatal("ocr made the input of fixer/d")
	}
	if domain.HasPluginInChain(versions, "ocr/c", "fixer") {
		t.Fatal("fixer made nothing ocr/c was made from")
	}
}

func TestFileInputComparesExtensionsWithoutCase(t *testing.T) {
	in := domain.FileInput{".pdf"}
	if !in.IsFileAccepted("books/A.PDF") {
		t.Fatal("A.PDF is a pdf")
	}
	if in.IsFileAccepted("books.pdf/readme") || in.IsFileAccepted("notes/a.md") {
		t.Fatal("a file of another type is accepted")
	}
}

func TestVersionTextLocatesSectionsPagesAndMoments(t *testing.T) {
	positioned := domain.NewVersionText(domain.TextVersion{Format: domain.TextFormatPositioned}, []domain.TextPart{
		{Text: "Intro ", Section: "One", Place: domain.TextPlace{Page: 0, Rects: []domain.Rect{{MaxX: 1}}}},
		{Text: "body ", Place: domain.TextPlace{Page: 1, Rects: []domain.Rect{{MaxX: 0.5}}}},
		{Text: "end", Section: "Two", Place: domain.TextPlace{Page: 1}},
	})
	if positioned.Text != "Intro body end" {
		t.Fatalf("text %q", positioned.Text)
	}
	if got := positioned.Locate(7); got != "One, page 2 of the file" {
		t.Fatalf("locate 7: %q", got)
	}
	if got := positioned.GetSections(); len(got) != 2 || got[1].Offset != 11 || got[1].Title != "Two" {
		t.Fatalf("sections %+v", got)
	}
	if got := positioned.GetNamesAt(11); len(got) != 1 || got[0] != "Two" {
		t.Fatalf("names at 11: %v", got)
	}
	boxes := positioned.GetBoxes(domain.ByteSpan{From: 6, To: 9})
	if len(boxes) != 1 || boxes[0].Page != 1 || boxes[0].From != 6 || boxes[0].To != 11 {
		t.Fatalf("boxes %+v", boxes)
	}

	timed := domain.NewVersionText(domain.TextVersion{Format: domain.TextFormatTimed}, []domain.TextPart{
		{Text: "hello ", Place: domain.TextPlace{Start: 0}},
		{Text: "world", Place: domain.TextPlace{Start: 3725 * time.Second}},
	})
	if got := timed.Locate(8); got != "1:02:05" {
		t.Fatalf("locate in timed text: %q", got)
	}
	if got := timed.GetBoxes(domain.ByteSpan{From: 0, To: 5}); got != nil {
		t.Fatalf("timed text has boxes: %+v", got)
	}
}

func TestFormatClock(t *testing.T) {
	for at, want := range map[time.Duration]string{0: "0:00", 65 * time.Second: "1:05", 3600 * time.Second: "1:00:00", -time.Second: "0:00"} {
		if got := domain.FormatClock(at); got != want {
			t.Errorf("FormatClock(%v) = %q, want %q", at, got, want)
		}
	}
}
