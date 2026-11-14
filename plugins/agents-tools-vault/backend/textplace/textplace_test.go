package textplace_test

import (
	"encoding/json"
	"errors"
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

func TestSelectorIsEmptyAndParsed(t *testing.T) {
	for _, tc := range []struct {
		selector textplace.Selector
		want     bool
	}{
		{selector: textplace.Selector{Page: 0, CFI: "", T: nil}, want: true},
		{selector: textplace.Selector{Page: 0, CFI: "", T: []float64{}}, want: true},
		{selector: textplace.Selector{Page: 1, CFI: "", T: nil}, want: false},
		{selector: textplace.Selector{Page: -1, CFI: "", T: nil}, want: false},
		{selector: textplace.Selector{Page: 0, CFI: "/6/2", T: nil}, want: false},
		{selector: textplace.Selector{Page: 0, CFI: "", T: []float64{1.5}}, want: false},
		{selector: textplace.Selector{Page: 0, CFI: "", T: []float64{1.5, 3.0}}, want: false},
	} {
		if got := tc.selector.IsEmpty(); got != tc.want {
			t.Errorf("%+v.IsEmpty() = %v, want %v", tc.selector, got, tc.want)
		}
	}

	empty, err := textplace.ParseSelector("")
	if err != nil || !empty.IsEmpty() {
		t.Errorf("ParseSelector(\"\") = %+v, %v, want empty selector and nil", empty, err)
	}

	invalid, err := textplace.ParseSelector("{bad")
	if err == nil || invalid.Page != 0 || !invalid.IsEmpty() {
		t.Errorf("ParseSelector(\"{bad\") = %+v, %v, want empty selector and error", invalid, err)
	}

	if got := textplace.GetSelector("{bad"); got.Page != 0 || !got.IsEmpty() {
		t.Errorf("GetSelector(\"{bad\") = %+v, want empty selector", got)
	}
	if got := textplace.GetSelector(`{"page":3,"cfi":"/6/4"}`); got.Page != 3 || got.CFI != "/6/4" {
		t.Errorf("GetSelector valid = %+v, want page 3 and cfi /6/4", got)
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
	start, end, err = textplace.FindPage(2, spans)
	if err != nil || start != 25 || end != 35 {
		t.Errorf("FindPage(2) = %d, %d, %v, want 25, 35, nil", start, end, err)
	}
	start, end, err = textplace.FindPage(3, spans)
	if err == nil || start != 0 || end != 0 || err.Error() != "the text has no page 3: its pages are 1 to 2" {
		t.Errorf("FindPage(3) = %d, %d, %v, want 0, 0, pages 1 to 2 error", start, end, err)
	}

	noPageSpans := []textplace.Span{
		{Start: 0, End: 10, Selector: textplace.Selector{Page: 0, CFI: "/6/2", T: nil}},
	}
	start, end, err = textplace.FindPage(1, noPageSpans)
	if err == nil || start != 0 || end != 0 || err.Error() != "the text has no pages; read it by start and length" {
		t.Errorf("FindPage on unpaged spans = %d, %d, %v, want 0, 0, no pages error", start, end, err)
	}

	unorderedSpans := []textplace.Span{
		{Start: 0, End: 5, Selector: textplace.Selector{Page: 0, CFI: "", T: nil}},
		{Start: 5, End: 10, Selector: textplace.Selector{Page: 4, CFI: "", T: nil}},
		{Start: 10, End: 15, Selector: textplace.Selector{Page: 2, CFI: "", T: nil}},
		{Start: 15, End: 20, Selector: textplace.Selector{Page: 5, CFI: "", T: nil}},
	}
	start, end, err = textplace.FindPage(3, unorderedSpans)
	if err == nil || start != 0 || end != 0 || err.Error() != "the text has no page 3: its pages are 2 to 5" {
		t.Errorf("FindPage(3, unordered) = %d, %d, %v, want 0, 0, pages 2 to 5 error", start, end, err)
	}

	negativePageSpans := []textplace.Span{
		{Start: 0, End: 4, Selector: textplace.Selector{Page: -2, CFI: "", T: nil}},
	}
	start, end, err = textplace.FindPage(1, negativePageSpans)
	if err == nil || start != 0 || end != 0 || err.Error() != "the text has no page 1: its pages are -2 to 0" {
		t.Errorf("FindPage(1, negative) = %d, %d, %v, want 0, 0, pages -2 to 0 error", start, end, err)
	}
}

func TestGetRangesJoinsTheSegmentsOfOnePage(t *testing.T) {
	spans, err := textplace.GetSpans(newSegments(), 0)
	if err != nil {
		t.Fatal(err)
	}
	ranges := textplace.GetRanges(spans, 0, 35)
	if len(ranges) != 2 ||
		ranges[0].Start != 0 || ranges[0].Length != 25 || ranges[0].Selector.Page != 1 ||
		ranges[1].Start != 25 || ranges[1].Length != 10 || ranges[1].Selector.Page != 2 {
		t.Errorf("GetRanges = %+v, want [0..25 page 1, 25..35 page 2]", ranges)
	}

	mixed := []textplace.Span{
		{Start: 0, End: 5, Selector: textplace.Selector{Page: 1, CFI: "", T: nil}},
		{Start: 0, End: 10, Selector: textplace.Selector{Page: 1, CFI: "", T: nil}},
		{Start: 10, End: 15, Selector: textplace.Selector{Page: 0, CFI: "", T: nil}},
		{Start: 15, End: 20, Selector: textplace.Selector{Page: 2, CFI: "", T: nil}},
		{Start: 20, End: 28, Selector: textplace.Selector{Page: 2, CFI: "", T: nil}},
		{Start: 28, End: 32, Selector: textplace.Selector{Page: 0, CFI: "/6/2", T: nil}},
		{Start: 32, End: 40, Selector: textplace.Selector{Page: 0, CFI: "/6/4", T: nil}},
	}
	got := textplace.GetRanges(mixed, 10, 35)
	if len(got) != 3 ||
		got[0].Start != 15 || got[0].Length != 13 || got[0].Selector.Page != 2 ||
		got[1].Start != 28 || got[1].Length != 4 || got[1].Selector.CFI != "/6/2" ||
		got[2].Start != 32 || got[2].Length != 3 || got[2].Selector.CFI != "/6/4" {
		t.Errorf("GetRanges(mixed, 10, 35) = %+v", got)
	}
}

func TestGetSpansRejectsASelectorThatIsNotJSON(t *testing.T) {
	spans, err := textplace.GetSpans(newSegments(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 3 ||
		spans[0].Start != 10 || spans[0].End != 22 ||
		spans[1].Start != 22 || spans[1].End != 35 ||
		spans[2].Start != 35 || spans[2].End != 45 {
		t.Errorf("GetSpans(offset=10) = %+v", spans)
	}

	segments := []domain.TextSegment{{Text: "Tide.", Selector: "page one"}}
	_, err = textplace.GetSpans(segments, 4)
	if err == nil {
		t.Fatal("GetSpans returned no error for an invalid selector")
	}
	if syntaxErr, ok := errors.AsType[*json.SyntaxError](err); !ok || syntaxErr == nil {
		t.Errorf("GetSpans error = %v, want wrapped *json.SyntaxError", err)
	}
}

func TestCheckReadRejectsAPageWithAStart(t *testing.T) {
	for _, tc := range []struct {
		page, start, length int
		wantErr             string
	}{
		{page: 0, start: 0, length: 0, wantErr: ""},
		{page: 1, start: 0, length: 10, wantErr: ""},
		{page: 0, start: 1, length: 10, wantErr: ""},
		{page: -1, start: 0, length: 0, wantErr: "page is -1: give a number of zero or more"},
		{page: 0, start: -1, length: 0, wantErr: "start is -1: give a number of zero or more"},
		{page: 0, start: 0, length: -1, wantErr: "length is -1: give a number of zero or more"},
		{page: 1, start: 1, length: 0, wantErr: "give a page, or a start and a length, not a page and a start"},
		{page: 2, start: 10, length: 0, wantErr: "give a page, or a start and a length, not a page and a start"},
	} {
		err := textplace.CheckRead(tc.page, tc.start, tc.length)
		if tc.wantErr == "" {
			if err != nil {
				t.Errorf("CheckRead(%d, %d, %d) = %v, want nil", tc.page, tc.start, tc.length, err)
			}
			continue
		}
		if err == nil || err.Error() != tc.wantErr {
			t.Errorf("CheckRead(%d, %d, %d) = %v, want %q", tc.page, tc.start, tc.length, err, tc.wantErr)
		}
	}
}
