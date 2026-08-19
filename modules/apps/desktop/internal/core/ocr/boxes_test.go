package ocr_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/ocr"
)

func TestBoxesComeBackAsTheyWereWritten(t *testing.T) {
	boxes := []ocr.Box{
		{Page: 0, Start: 0, Length: 5, MinX: 0, MinY: 0.25, MaxX: 0.5, MaxY: 0.75},
		{Page: 1, Start: 25, Length: 7, MinX: 0.125, MinY: 0.5, MaxX: 1, MaxY: 1},
		{Page: 17, Start: 4096, Length: 1, MinX: 0.1, MinY: 0.2, MaxX: 0.3, MaxY: 0.4},
	}
	raw := ocr.Pack(boxes)

	if len(raw) != 3*28 {
		t.Errorf("packed %d bytes for three boxes, want %d", len(raw), 3*28)
	}
	if got := ocr.Unpack(raw); !reflect.DeepEqual(got, boxes) {
		t.Errorf("unpacked %+v, want %+v", got, boxes)
	}
}

func TestATornTailGivesBackTheWholeRecords(t *testing.T) {
	// A run stopped part way through writing leaves a record half written.
	// What was written whole is still a box.
	boxes := []ocr.Box{
		{Page: 0, Start: 0, Length: 5, MaxX: 0.5, MaxY: 0.5},
		{Page: 0, Start: 6, Length: 4, MaxX: 0.75, MaxY: 0.5},
	}
	raw := ocr.Pack(boxes)

	got := ocr.Unpack(raw[:len(raw)-9])
	if len(got) != 1 {
		t.Fatalf("unpacked %d boxes from one whole record and part of another", len(got))
	}
	if !reflect.DeepEqual(got[0], boxes[0]) {
		t.Errorf("unpacked %+v, want %+v", got[0], boxes[0])
	}
	if left := ocr.Unpack(raw[:12]); len(left) != 0 {
		t.Errorf("unpacked %d boxes from less than one record", len(left))
	}
}
