package domain_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func TestASpanIsEmptyWhenItEndsWhereItBeginsOrBefore(t *testing.T) {
	for _, one := range []struct {
		from, to int
		empty    bool
		length   int
	}{
		{3, 9, false, 6},
		{3, 4, false, 1},
		{3, 3, true, 0},
		{9, 3, true, -6},
		{0, 0, true, 0},
	} {
		bytes := domain.ByteSpan{From: one.from, To: one.to}
		if bytes.Empty() != one.empty {
			t.Errorf("bytes %d..%d: Empty is %v, want %v", one.from, one.to, bytes.Empty(), one.empty)
		}
		if bytes.Len() != one.length {
			t.Errorf("bytes %d..%d: Len is %d, want %d", one.from, one.to, bytes.Len(), one.length)
		}
		units := domain.UnitSpan{From: one.from, To: one.to}
		if units.Empty() != one.empty {
			t.Errorf("units %d..%d: Empty is %v, want %v", one.from, one.to, units.Empty(), one.empty)
		}
		if units.Len() != one.length {
			t.Errorf("units %d..%d: Len is %d, want %d", one.from, one.to, units.Len(), one.length)
		}
	}
}

func TestMostHighlightsIsEight(t *testing.T) {
	if domain.MostHighlights != 8 {
		t.Errorf("MostHighlights is %d, want 8", domain.MostHighlights)
	}
}
