package ocr_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/ocr"
)

func TestPartsComeBackAsTheyWereWritten(t *testing.T) {
	parts := []ocr.Part{
		{Start: 0, Length: 19, Depth: 0},
		{Start: 421, Length: 33, Depth: 1},
		{Start: 4096, Length: 1, Depth: 2},
	}
	raw := ocr.Pack(parts)

	if len(raw) != 3*12 {
		t.Errorf("packed %d bytes for three parts, want %d", len(raw), 3*12)
	}
	if got := ocr.Unpack(raw); !reflect.DeepEqual(got, parts) {
		t.Errorf("unpacked %+v, want %+v", got, parts)
	}
}

func TestATornTailGivesBackTheWholeParts(t *testing.T) {
	// A run stopped part way through writing leaves a record half written.
	// What was written whole is still a part.
	parts := []ocr.Part{
		{Start: 0, Length: 19, Depth: 0},
		{Start: 421, Length: 33, Depth: 1},
	}
	raw := ocr.Pack(parts)

	got := ocr.Unpack(raw[:len(raw)-5])
	if len(got) != 1 {
		t.Fatalf("unpacked %d parts from one whole record and part of another", len(got))
	}
	if !reflect.DeepEqual(got[0], parts[0]) {
		t.Errorf("unpacked %+v, want %+v", got[0], parts[0])
	}
	if left := ocr.Unpack(raw[:7]); len(left) != 0 {
		t.Errorf("unpacked %d parts from less than one record", len(left))
	}
}
