package service_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/service"
)

// countingTexts is text versions that count the pages of text read through them.
type countingTexts struct {
	host.Texts
	reads *atomic.Int64
}

func (c countingTexts) With(id domain.VaultID) host.Texts {
	return countingTexts{Texts: c.Texts.With(id), reads: c.reads}
}

func (c countingTexts) ReadText(ctx context.Context, path string, id domain.TextVersionID, cursor string) (host.TextPage, error) {
	c.reads.Add(1)
	return c.Texts.ReadText(ctx, path, id, cursor)
}

// positionTools is the window tools over a vault of a guide and a book whose text reads are counted.
type positionTools struct {
	windowTools
	texts *hosttest.Texts
	reads *atomic.Int64
}

func newPositionTools(t *testing.T, bookSegments []domain.TextSegment) positionTools {
	t.Helper()
	vault := hosttest.New(testVault)
	vault.File(guidePath, pdfBytes)
	vault.File(bookPath, epubBytes)
	texts := hosttest.NewTexts(testVault, readerID)
	texts.File(guidePath, []byte(pdfBytes))
	texts.File(bookPath, []byte(epubBytes))
	putText(t, texts, guidePath, "page-text", []byte(pdfBytes), []domain.TextSegment{
		{Text: "The marsh at dawn. ", Selector: `{"page":1}`},
		{Text: "Index of birds. ", Selector: `{"page":120}`},
	})
	putText(t, texts, bookPath, "book-text", []byte(epubBytes), bookSegments)

	reads := &atomic.Int64{}
	window := hosttest.NewWindow(vault)
	registry := tools.New()
	manager := service.NewManager(clock.NewFake(time.Unix(1_700_000_000, 0)))
	on := service.Host{Vault: vault, Window: window, Texts: countingTexts{Texts: texts, reads: reads}}
	if err := service.RegisterTools(registry, manager, on); err != nil {
		t.Fatal(err)
	}
	return positionTools{windowTools: windowTools{window: window, calls: registry}, texts: texts, reads: reads}
}

// getRealBookSegments is a book in the form the EPUB reader writes it: an empty segment opens each
// section and holds its heading, and the text segments after it have none.
func getRealBookSegments() []domain.TextSegment {
	return []domain.TextSegment{
		{Text: "", Heading: "The Marsh at Dawn", Selector: `{"cfi":"epubcfi(/6/4[ch01]!/4[body01]/2/1:0)"}`},
		{Text: "The marsh wakes slowly. ", Selector: `{"cfi":"epubcfi(/6/4[ch01]!/4[body01]/2/1:0)"}`},
		{Text: "Mist lies on the water. ", Selector: `{"cfi":"epubcfi(/6/4[ch01]!/4[body01]/8/1:0)"}`},
		{Text: "", Heading: "Herons in Winter", Selector: `{"cfi":"epubcfi(/6/6!/4/2/1:0)"}`},
		{Text: "In winter the herons stay. ", Selector: `{"cfi":"epubcfi(/6/6!/4/2/1:0)"}`},
		{Text: "They fish under the ice. ", Selector: `{"cfi":"epubcfi(/6/6!/4/12/1:0)"}`},
	}
}

func TestWindowTabsGetReadsTheChapterOfABookWhoseHeadingsOpenEmptySegments(t *testing.T) {
	for name, tc := range map[string]struct{ cfi, want string }{
		"a CFI inside a chapter, with assertions": {`epubcfi(/6/4[ch01]!/4[body01]/8/1:12)`, `chapter "The Marsh at Dawn"`},
		"the start of a chapter":                  {`epubcfi(/6/6!/4/2/1:0)`, `chapter "Herons in Winter"`},
		"a range CFI":                             {`epubcfi(/6/6!/4,/12/1:0,/12/1:5)`, `chapter "Herons in Winter"`},
		"a CFI before every segment":              {`epubcfi(/6/2!/4/2/1:0)`, ""},
		"a CFI past the end":                      {`epubcfi(/6/8!/4/2/1:0)`, `chapter "Herons in Winter"`},
	} {
		t.Run(name, func(t *testing.T) {
			w := newPositionTools(t, getRealBookSegments())
			out := w.readTabs(t, []host.OpenTab{{ID: "tab-1", Path: bookPath, State: `{"cfi":"` + tc.cfi + `"}`}}, "tab-1")
			if len(out.Tabs) != 1 || out.Tabs[0].Location != tc.want {
				t.Errorf("answered %+v, want the tab at %q", out.Tabs, tc.want)
			}
		})
	}
}

func TestWindowTabsGetReadsEachTextVersionOnceAcrossTabsAndCalls(t *testing.T) {
	tabs := []host.OpenTab{
		{ID: "tab-1", Path: guidePath, State: `{"page":4}`},
		{ID: "tab-2", Path: guidePath, State: `{"page":7}`},
		{ID: "tab-3", Path: bookPath, State: `{"cfi":"epubcfi(/6/6!/4/2/1:0)"}`},
	}
	single := newPositionTools(t, getRealBookSegments())
	single.readTabs(t, []host.OpenTab{tabs[0], tabs[2]}, "tab-1")
	perFile := single.reads.Load()
	if perFile == 0 {
		t.Fatal("no text was read")
	}

	w := newPositionTools(t, getRealBookSegments())
	out := w.readTabs(t, tabs, "tab-1")
	if got := w.reads.Load(); got != perFile {
		t.Errorf("two tabs of one file read %d pages, want the %d of one tab of each file", got, perFile)
	}
	if out.Tabs[0].Location != "page 4 of 120" || out.Tabs[1].Location != "page 7 of 120" || out.Tabs[2].Location != `chapter "Herons in Winter"` {
		t.Errorf("answered %+v", out.Tabs)
	}
	w.readTabs(t, tabs, "tab-1")
	w.readTabs(t, tabs, "tab-1")
	if got := w.reads.Load(); got != perFile {
		t.Errorf("three calls read %d pages, want the %d of the first", got, perFile)
	}

	putText(t, w.texts, guidePath, "page-text", []byte(pdfBytes), []domain.TextSegment{
		{Text: "A new edition. ", Selector: `{"page":200}`},
	})
	out = w.readTabs(t, tabs, "tab-1")
	if out.Tabs[0].Location != "page 4 of 200" {
		t.Errorf("after a new text version the tab is at %q, want page 4 of 200", out.Tabs[0].Location)
	}
}
