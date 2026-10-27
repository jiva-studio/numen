package host_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

func TestASelectorEncodesItsDocumentedKeysAndReadsBack(t *testing.T) {
	selector := host.Selector{T: []float64{1.5, 4}, Page: 3, Boxes: [][4]float32{{0.1, 0.2, 0.5, 0.25}}}.Encode()
	if selector != `{"t":[1.5,4],"page":3,"boxes":[[0.1,0.2,0.5,0.25]]}` {
		t.Fatalf("encoded %s", selector)
	}
	read, err := host.ParseSelector(selector)
	if err != nil || read.Page != 3 || read.T[1] != 4 || read.Boxes[0][2] != 0.5 {
		t.Fatalf("read %+v (%v)", read, err)
	}
	if _, err := host.ParseSelector("page 3"); err == nil {
		t.Fatal("a selector that is not JSON was read")
	}
	if empty, err := host.ParseSelector(""); err != nil || empty.Page != 0 {
		t.Fatalf("an empty selector read %+v (%v)", empty, err)
	}
}

func TestGetBoxesReturnsTheBoxesOfTheSegmentsARunOverlaps(t *testing.T) {
	text := domain.NewVersionText(domain.TextVersion{}, []domain.TextSegment{
		{Text: "alpha ", Selector: host.Selector{Page: 1, Boxes: [][4]float32{{0.1, 0.1, 0.2, 0.2}}}.Encode()},
		{Text: "beta ", Selector: host.Selector{Page: 2, Boxes: [][4]float32{{0.3, 0.3, 0.4, 0.4}}}.Encode()},
		{Text: "gamma", Selector: host.Selector{T: []float64{1, 2}}.Encode()},
	})

	boxes := host.GetBoxes(text, domain.ByteRange{Start: 3, End: 16})

	if len(boxes) != 2 || boxes[0].Page != 0 || boxes[1].Page != 1 || boxes[1].Start != 6 || boxes[1].End != 11 || boxes[1].MinX != 0.3 {
		t.Errorf("boxes = %+v, want one on each page, none for the segment without a page", boxes)
	}
	if got := host.GetBoxes(text, domain.ByteRange{Start: 4, End: 4}); got != nil {
		t.Errorf("an empty run has boxes %+v", got)
	}
}
