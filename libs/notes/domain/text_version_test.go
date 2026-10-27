package domain_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

var epoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func isAllIntact(domain.TextVersion) bool { return true }

func TestFindLiveVersionReturnsTheNewestValidVersion(t *testing.T) {
	versions := []domain.TextVersion{
		{Path: "a.pdf", PluginID: "reader", Key: "a", SourceHash: "file-1", Hash: "ha", CreateTime: epoch},
		{Path: "a.pdf", PluginID: "fixer", Key: "b", Source: "reader/a", SourceHash: "ha", Hash: "hb", CreateTime: epoch.Add(time.Minute)},
	}
	got, ok := domain.FindLiveVersion(versions, "file-1", isAllIntact)
	if !ok || got.GetID() != "fixer/b" {
		t.Fatalf("got %q (%v), want fixer/b", got.GetID(), ok)
	}
}

func TestFindLiveVersionBreaksATieByListOrder(t *testing.T) {
	versions := []domain.TextVersion{
		{PluginID: "p", Key: "first", SourceHash: "f", CreateTime: epoch},
		{PluginID: "p", Key: "second", SourceHash: "f", CreateTime: epoch},
	}
	got, _ := domain.FindLiveVersion(versions, "f", isAllIntact)
	if got.Key != "second" {
		t.Fatalf("got %q, want the later entry", got.Key)
	}
}

func TestFindLiveVersionSkipsAVersionWhoseFileChanged(t *testing.T) {
	versions := []domain.TextVersion{
		{PluginID: "reader", Key: "a", SourceHash: "f", Hash: "ha", CreateTime: epoch},
		{PluginID: "fixer", Key: "b", Source: "reader/a", SourceHash: "ha", Hash: "hb", CreateTime: epoch.Add(time.Minute)},
	}
	got, ok := domain.FindLiveVersion(versions, "f", func(v domain.TextVersion) bool { return v.PluginID != "fixer" })
	if !ok || got.GetID() != "reader/a" {
		t.Fatalf("got %q (%v), want the version before the changed one", got.GetID(), ok)
	}
}

func TestIsChainValidFailsWhenTheSourceChanged(t *testing.T) {
	versions := []domain.TextVersion{
		{PluginID: "reader", Key: "a", SourceHash: "file-1", Hash: "ha-2"},
		{PluginID: "fixer", Key: "b", Source: "reader/a", SourceHash: "ha-1", Hash: "hb"},
	}
	if !domain.IsChainValid(versions, "reader/a", "file-1") {
		t.Fatal("the version from the file is valid")
	}
	if domain.IsChainValid(versions, "fixer/b", "file-1") {
		t.Fatal("a version made from other text of its source is valid")
	}
}

func TestIsChainValidFailsWhenTheFileChanged(t *testing.T) {
	versions := []domain.TextVersion{{PluginID: "reader", Key: "a", SourceHash: "file-1"}}
	if domain.IsChainValid(versions, "reader/a", "file-2") {
		t.Fatal("a version of other bytes of the file is valid")
	}
}

func TestIsChainValidFailsOnAMissingSourceAndOnACycle(t *testing.T) {
	missing := []domain.TextVersion{{PluginID: "fixer", Key: "b", Source: "reader/a"}}
	if domain.IsChainValid(missing, "fixer/b", "f") {
		t.Fatal("a version whose source is gone is valid")
	}
	cycle := []domain.TextVersion{
		{PluginID: "p", Key: "x", Source: "p/y", SourceHash: "hy", Hash: "hx"},
		{PluginID: "p", Key: "y", Source: "p/x", SourceHash: "hx", Hash: "hy"},
	}
	if domain.IsChainValid(cycle, "p/x", "f") {
		t.Fatal("a version reaching itself is valid")
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

func TestVersionTextLocatesHeadingsAndSelectors(t *testing.T) {
	text := domain.NewVersionText(domain.TextVersion{Kind: "recognition"}, []domain.TextSegment{
		{Text: "Intro ", Heading: "One", Selector: `{"page":1}`},
		{Text: "body ", Selector: `{"page":2}`},
		{Text: "end", Heading: "Two"},
	})
	if text.Text != "Intro body end" {
		t.Fatalf("text %q", text.Text)
	}
	if got := text.Locate(7); got != "One" {
		t.Fatalf("locate 7: %q", got)
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
	if got := text.GetHeadingsAt(11); len(got) != 1 || got[0] != "Two" {
		t.Fatalf("headings at 11: %v", got)
	}
	if got := len(text.GetSegments()); got != 3 {
		t.Fatalf("%d segments", got)
	}
}
