package textlayer_test

import (
	"errors"
	"math"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/internal/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/textlayer"
)

// TestMain makes the folder the pools of the tests keep the compiled PDFium in.
func TestMain(m *testing.M) {
	os.Exit(testsupport.Run(m))
}

func TestGetBoxesReadsTheBoxesOfTheSegmentsARunCovers(t *testing.T) {
	if sel, err := textlayer.ReadPageSelector(""); err != nil || sel.Page != 0 || len(sel.Boxes) != 0 {
		t.Fatalf("ReadPageSelector(\"\") = %+v, %v; want empty PageSelector, nil", sel, err)
	}

	version := domain.NewVersionText(domain.TextVersion{}, []domain.TextSegment{
		{Text: "alpha", Selector: `{"page":1,"boxes":[[0.1,0.1,0.2,0.2]]}`},
		{Text: "beta\n", Selector: `{"page":2,"boxes":[[0.1,0.2,0.3,0.4]]}`},
		{Text: "no", Selector: `{"page":0,"boxes":[[0.1,0.1,0.2,0.2]]}`},
		{Text: "neg", Selector: `{"page":-1,"boxes":[[0.1,0.1,0.2,0.2]]}`},
		{Text: "bad", Selector: `{"page":1,"boxes":[[0.1,0.2,0.3,0.4]],"x":}`},
		{Text: "gamma", Selector: `{"page":3,"boxes":[[0.7,0.7,0.8,0.8]]}`},
	})

	if got := textlayer.GetBoxes(version, pdf.ByteRange{Start: 2, End: 2}); len(got) != 0 {
		t.Errorf("empty range boxes = %+v, want none", got)
	}
	if got := textlayer.GetBoxes(version, pdf.ByteRange{Start: 3, End: 2}); len(got) != 0 {
		t.Errorf("inverted range boxes = %+v, want none", got)
	}

	boxes := textlayer.GetBoxes(version, pdf.ByteRange{Start: 5, End: 18})
	want := pdf.HighlightBox{
		Page:      1,
		ByteRange: pdf.ByteRange{Start: 5, End: 10},
		Rect:      pdf.Rect{MinX: 0.1, MinY: 0.2, MaxX: 0.3, MaxY: 0.4},
	}
	if len(boxes) != 1 || boxes[0] != want {
		t.Errorf("boxes = %+v, want [%+v]", boxes, want)
	}
}

func TestPageSegmentsSplitBoxesAndSectionsAcrossPages(t *testing.T) {
	book := &pdf.Book{
		Text: "Alpha betaGamma deltaOmega",
		Pages: []pdf.Page{
			{Offset: 0},
			{Offset: 10},
			{Offset: 21},
		},
		Parts: []pdf.Part{
			{Title: "Intro", Offset: 0},
			{Title: "Mid", Offset: 3},
			{Title: "PageTwo", Offset: 10},
			{Title: "Tail", Offset: 19},
		},
	}
	boxes := []pdf.HighlightBox{
		{Page: 0, ByteRange: pdf.ByteRange{Start: 0, End: 0}, Rect: pdf.Rect{MinX: 0.9, MinY: 0.9, MaxX: 1, MaxY: 1}},
		{Page: 0, ByteRange: pdf.ByteRange{Start: 0, End: 5}, Rect: pdf.Rect{MinX: 0.1, MinY: 0.2, MaxX: 0.3, MaxY: 0.4}},
		{Page: 0, ByteRange: pdf.ByteRange{Start: 6, End: 10}, Rect: pdf.Rect{MinX: 0.5, MinY: 0.2, MaxX: 0.7, MaxY: 0.4}},
		{Page: 1, ByteRange: pdf.ByteRange{Start: 10, End: 15}, Rect: pdf.Rect{MinX: 0.1, MinY: 0.5, MaxX: 0.4, MaxY: 0.6}},
	}

	got, err := textlayer.CutPages(book, 0, 2, boxes)
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.TextSegment{
		{Text: "", Heading: "Intro", Selector: `{"page":1}`},
		{Text: "Alp", Heading: "", Selector: `{"page":1,"boxes":[[0.1,0.2,0.3,0.4]]}`},
		{Text: "", Heading: "Mid", Selector: `{"page":1}`},
		{Text: "ha", Heading: "", Selector: `{"page":1,"boxes":[[0.1,0.2,0.3,0.4]]}`},
		{Text: " ", Heading: "", Selector: `{"page":1}`},
		{Text: "beta", Heading: "", Selector: `{"page":1,"boxes":[[0.5,0.2,0.7,0.4]]}`},
		{Text: "", Heading: "PageTwo", Selector: `{"page":2}`},
		{Text: "Gamma", Heading: "", Selector: `{"page":2,"boxes":[[0.1,0.5,0.4,0.6]]}`},
		{Text: " del", Heading: "", Selector: `{"page":2}`},
		{Text: "", Heading: "Tail", Selector: `{"page":2}`},
		{Text: "ta", Heading: "", Selector: `{"page":2}`},
	}
	if !slices.Equal(got, want) {
		t.Errorf("CutPages(0, 2) = %+v, want %+v", got, want)
	}

	withStalePageZero := append([]pdf.HighlightBox{
		{Page: 0, ByteRange: pdf.ByteRange{Start: 10, End: 15}, Rect: pdf.Rect{MinX: 0.8, MinY: 0.8, MaxX: 0.9, MaxY: 0.9}},
	}, boxes...)
	pageTwoOnly, err := textlayer.CutPages(book, 1, 2, withStalePageZero)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(pageTwoOnly, want[6:]) {
		t.Errorf("CutPages(1, 2) = %+v, want %+v", pageTwoOnly, want[6:])
	}

	nanBoxes := []pdf.HighlightBox{
		{Page: 0, ByteRange: pdf.ByteRange{Start: 0, End: 5}, Rect: pdf.Rect{MinX: float32(math.NaN()), MinY: 0, MaxX: 1, MaxY: 1}},
	}
	if _, err := textlayer.CutPages(book, 0, 1, nanBoxes); err == nil {
		t.Error("CutPages succeeded with NaN box coordinate")
	}
	if _, err := textlayer.GetSegments(book, func([]int) ([]pdf.HighlightBox, error) {
		return nanBoxes, nil
	}); err == nil {
		t.Error("GetSegments succeeded with NaN box coordinate")
	}
}

func TestGetSegmentsReadsPagesInBatches(t *testing.T) {
	pages := make([]pdf.Page, 20)
	var text strings.Builder
	for i := range pages {
		pages[i] = pdf.Page{Offset: text.Len()}
		text.WriteString("x")
	}
	book := &pdf.Book{Text: text.String(), Pages: pages}
	var batches [][]int
	segments, err := textlayer.GetSegments(book, func(wanted []int) ([]pdf.HighlightBox, error) {
		batches = append(batches, slices.Clone(wanted))
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(segments) != 20 || len(batches) != 2 || len(batches[0]) != 16 || len(batches[1]) != 4 {
		t.Errorf("GetSegments batches = %v (%d segments), want [16, 4]", batches, len(segments))
	}

	wantErr := errors.New("box read failed")
	if _, err := textlayer.GetSegments(book, func([]int) ([]pdf.HighlightBox, error) {
		return nil, wantErr
	}); !errors.Is(err, wantErr) {
		t.Errorf("GetSegments error = %v, want %v", err, wantErr)
	}
}

func TestPagesOfTheLayerJoinIntoTheWholeText(t *testing.T) {
	raw := testsupport.ReadFixture(t, "outline.pdf")
	workers := testsupport.NewWorkers(t)
	book, err := workers.Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	all := make([]int, len(book.Pages))
	for i := range all {
		all[i] = i
	}
	boxes, err := book.FindHighlights(workers, raw, all)
	if err != nil {
		t.Fatal(err)
	}

	var joined strings.Builder
	sections := 0
	for page := range book.Pages {
		layerParts, err := textlayer.CutPages(book, page, page+1, boxes)
		if err != nil {
			t.Fatal(err)
		}
		for _, part := range layerParts {
			if selector, err := textlayer.ReadPageSelector(part.Selector); err != nil || selector.Page != page+1 {
				t.Errorf("a segment of page %d has selector %s", page, part.Selector)
			}
			if part.Heading != "" {
				sections++
			}
			joined.WriteString(part.Text)
		}
	}

	if joined.String() != book.Text {
		t.Errorf("the pages join into %q, want %q", joined.String(), book.Text)
	}
	if sections != len(book.Parts) {
		t.Errorf("the pages open %d sections, want %d", sections, len(book.Parts))
	}
}
