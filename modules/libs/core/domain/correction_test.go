package domain_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func TestCorrectionsComeBackAsTheyWereWritten(t *testing.T) {
	lines := []domain.CorrectionLine{
		{Number: 0, Text: "Śrī Jayadeva Gosvāmī"},
		{Number: 7, Text: ""},
		{Number: 4096, Text: "the last line."},
	}
	raw := domain.PackCorrections(lines)

	want := 3*8 + len("Śrī Jayadeva Gosvāmī") + len("the last line.")
	if len(raw) != want {
		t.Errorf("packed %d bytes, want %d", len(raw), want)
	}
	if got := domain.UnpackCorrections(raw); !reflect.DeepEqual(got, lines) {
		t.Errorf("unpacked %+v, want %+v", got, lines)
	}
}

func TestATornTailGivesBackTheWholeCorrections(t *testing.T) {
	// A run stopped part way through writing leaves a record half written. What
	// was written whole is still a correction.
	lines := []domain.CorrectionLine{
		{Number: 0, Text: "Śrī Jayadeva Gosvāmī"},
		{Number: 1, Text: "and the second line."},
	}
	raw := domain.PackCorrections(lines)

	got := domain.UnpackCorrections(raw[:len(raw)-5])
	if len(got) != 1 {
		t.Fatalf("unpacked %d corrections from one whole record and part of another", len(got))
	}
	if !reflect.DeepEqual(got[0], lines[0]) {
		t.Errorf("unpacked %+v, want %+v", got[0], lines[0])
	}
	if left := domain.UnpackCorrections(raw[:6]); len(left) != 0 {
		t.Errorf("unpacked %d corrections from less than one header", len(left))
	}
}

func TestAppendedRunsReadAsOneFile(t *testing.T) {
	first := []domain.CorrectionLine{{Number: 0, Text: "Śrī Jayadeva Gosvāmī"}}
	second := []domain.CorrectionLine{{Number: 3, Text: ""}, {Number: 4, Text: "the last line."}}

	raw := append(domain.PackCorrections(first), domain.PackCorrections(second)...)

	want := append(append([]domain.CorrectionLine{}, first...), second...)
	if got := domain.UnpackCorrections(raw); !reflect.DeepEqual(got, want) {
		t.Errorf("unpacked %+v, want %+v", got, want)
	}
}
