package textlayer_test

import (
	"os"
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
	version := domain.NewVersionText(domain.TextVersion{}, []domain.TextSegment{
		{Text: "hello", Selector: `{"page":1,"boxes":[[0.1,0.2,0.3,0.4]]}`},
	})

	boxes := textlayer.GetBoxes(version, pdf.ByteRange{Start: 0, End: 5})

	if len(boxes) != 1 || boxes[0].MinX != 0.1 || boxes[0].MinY != 0.2 || boxes[0].MaxX != 0.3 || boxes[0].MaxY != 0.4 {
		t.Errorf("boxes = %+v, want [0.1, 0.2, 0.3, 0.4]", boxes)
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
	boxes, err := book.Highlights(workers, raw, all)
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
