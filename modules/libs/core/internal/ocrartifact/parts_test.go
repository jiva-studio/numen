package ocrartifact_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/ocrartifact"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func TestPartsComeBackAsTheyWereSaved(t *testing.T) {
	parts := []domain.OCRPart{
		{Start: 0, Length: 19, Depth: 0},
		{Start: 421, Length: 33, Depth: 1},
		{Start: 4096, Length: 1, Depth: 2},
	}
	raw := ocrartifact.Pack(parts)

	if len(raw) != 3*12 {
		t.Errorf("packed %d bytes for three parts, want %d", len(raw), 3*12)
	}
	if got := ocrartifact.Unpack(raw); !reflect.DeepEqual(got, parts) {
		t.Errorf("unpacked %+v, want %+v", got, parts)
	}
}

func TestATornTailProducesBackTheWholeParts(t *testing.T) {
	// A run stopped part way through writing leaves a record half written.
	// What was written whole is still a part.
	parts := []domain.OCRPart{
		{Start: 0, Length: 19, Depth: 0},
		{Start: 421, Length: 33, Depth: 1},
	}
	raw := ocrartifact.Pack(parts)

	got := ocrartifact.Unpack(raw[:len(raw)-5])
	if len(got) != 1 {
		t.Fatalf("unpacked %d parts from one whole record and part of another", len(got))
	}
	if !reflect.DeepEqual(got[0], parts[0]) {
		t.Errorf("unpacked %+v, want %+v", got[0], parts[0])
	}
	if remaining := ocrartifact.Unpack(raw[:7]); len(remaining) != 0 {
		t.Errorf("unpacked %d parts from less than one record", len(remaining))
	}
}
