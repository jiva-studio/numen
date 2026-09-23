package correction_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/correction"
)

func TestCorrectionsComeBackAsTheyWereWritten(t *testing.T) {
	lines := []correction.Line{
		{Number: 0, Text: "Śrī Jayadeva Gosvāmī"},
		{Number: 7, Text: ""},
		{Number: 4096, Text: "the last line."},
	}
	raw := correction.Pack(lines)

	want := 3*8 + len("Śrī Jayadeva Gosvāmī") + len("the last line.")
	if len(raw) != want {
		t.Errorf("packed %d bytes, want %d", len(raw), want)
	}
	if got := correction.Unpack(raw); !reflect.DeepEqual(got, lines) {
		t.Errorf("unpacked %+v, want %+v", got, lines)
	}
}

func TestATornTailGivesBackTheWholeCorrections(t *testing.T) {
	// A run stopped part way through writing leaves a record half written. What
	// was written whole is still a correction.
	lines := []correction.Line{
		{Number: 0, Text: "Śrī Jayadeva Gosvāmī"},
		{Number: 1, Text: "and the second line."},
	}
	raw := correction.Pack(lines)

	got := correction.Unpack(raw[:len(raw)-5])
	if len(got) != 1 {
		t.Fatalf("unpacked %d corrections from one whole record and part of another", len(got))
	}
	if !reflect.DeepEqual(got[0], lines[0]) {
		t.Errorf("unpacked %+v, want %+v", got[0], lines[0])
	}
	if left := correction.Unpack(raw[:6]); len(left) != 0 {
		t.Errorf("unpacked %d corrections from less than one header", len(left))
	}
}

func TestAppendedRunsReadAsOneFile(t *testing.T) {
	first := []correction.Line{{Number: 0, Text: "Śrī Jayadeva Gosvāmī"}}
	second := []correction.Line{{Number: 3, Text: ""}, {Number: 4, Text: "the last line."}}

	raw := append(correction.Pack(first), correction.Pack(second)...)

	want := append(append([]correction.Line{}, first...), second...)
	if got := correction.Unpack(raw); !reflect.DeepEqual(got, want) {
		t.Errorf("unpacked %+v, want %+v", got, want)
	}
}
