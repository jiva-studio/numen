package host_test

import (
	"testing"

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
