package fixes_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/fixes"
)

func TestCorrectionsComeBackAsTheyWereWritten(t *testing.T) {
	lines := []fixes.Line{
		{At: 0, Text: "Śrī Jayadeva Gosvāmī"},
		{At: 7, Text: ""},
		{At: 4096, Text: "the last line."},
	}
	raw := fixes.Pack(lines)

	want := 3*8 + len("Śrī Jayadeva Gosvāmī") + len("the last line.")
	if len(raw) != want {
		t.Errorf("packed %d bytes, want %d", len(raw), want)
	}
	if got := fixes.Unpack(raw); !reflect.DeepEqual(got, lines) {
		t.Errorf("unpacked %+v, want %+v", got, lines)
	}
}

func TestATornTailGivesBackTheWholeCorrections(t *testing.T) {
	// A run stopped part way through writing leaves a record half written. What
	// was written whole is still a correction.
	lines := []fixes.Line{
		{At: 0, Text: "Śrī Jayadeva Gosvāmī"},
		{At: 1, Text: "and the second line."},
	}
	raw := fixes.Pack(lines)

	got := fixes.Unpack(raw[:len(raw)-5])
	if len(got) != 1 {
		t.Fatalf("unpacked %d corrections from one whole record and part of another", len(got))
	}
	if !reflect.DeepEqual(got[0], lines[0]) {
		t.Errorf("unpacked %+v, want %+v", got[0], lines[0])
	}
	if left := fixes.Unpack(raw[:6]); len(left) != 0 {
		t.Errorf("unpacked %d corrections from less than one header", len(left))
	}
}

func TestAppendedRunsReadAsOneFile(t *testing.T) {
	first := []fixes.Line{{At: 0, Text: "Śrī Jayadeva Gosvāmī"}}
	second := []fixes.Line{{At: 3, Text: ""}, {At: 4, Text: "the last line."}}

	raw := append(fixes.Pack(first), fixes.Pack(second)...)

	want := append(append([]fixes.Line{}, first...), second...)
	if got := fixes.Unpack(raw); !reflect.DeepEqual(got, want) {
		t.Errorf("unpacked %+v, want %+v", got, want)
	}
}
