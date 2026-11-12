package textplace_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/textplace"
)

func newSegments() []domain.TextSegment {
	return []domain.TextSegment{
		{Text: "First tide. ", Selector: `{"page":1}`},
		{Text: "Second tide. ", Selector: `{"page":1}`},
		{Text: "Low water.", Selector: `{"page":2}`},
	}
}

func TestFindPageReturnsTheBytesOfThePage(t *testing.T) {
	spans, err := textplace.GetSpans(newSegments(), 0)
	if err != nil {
		t.Fatal(err)
	}
	start, end, err := textplace.FindPage(1, spans)
	if err != nil || start != 0 || end != 25 {
		t.Errorf("FindPage(1) = %d, %d, %v, want 0, 25, nil", start, end, err)
	}
	if _, _, err := textplace.FindPage(3, spans); err == nil {
		t.Error("FindPage(3) returned no error for a text of two pages")
	}
}

func TestGetRangesJoinsTheSegmentsOfOnePage(t *testing.T) {
	spans, err := textplace.GetSpans(newSegments(), 0)
	if err != nil {
		t.Fatal(err)
	}
	ranges := textplace.GetRanges(spans, 0, 35)
	if len(ranges) != 2 || ranges[0].Length != 25 || ranges[1].Selector.Page != 2 {
		t.Errorf("GetRanges = %+v, want one range for page 1 of 25 bytes and one for page 2", ranges)
	}
}

func TestGetSpansRejectsASelectorThatIsNotJSON(t *testing.T) {
	segments := []domain.TextSegment{{Text: "Tide.", Selector: "page one"}}
	if _, err := textplace.GetSpans(segments, 0); err == nil {
		t.Error("GetSpans returned no error for an invalid selector")
	}
}

func TestCheckReadRejectsAPageWithAStart(t *testing.T) {
	if err := textplace.CheckRead(2, 10, 0); err == nil {
		t.Error("CheckRead accepted a page together with a start")
	}
}
