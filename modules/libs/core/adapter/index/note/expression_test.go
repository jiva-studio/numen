package note

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func TestAMarkedNameIsReadBackAsTheNameAndTheRunsInIt(t *testing.T) {
	for _, c := range []struct {
		marked string
		name   string
		at     []domain.UnitSpan
	}{
		{"Entropy", "Entropy", nil},
		{"\x02Entropy\x03", "Entropy", []domain.UnitSpan{{From: 0, To: 7}}},
		{"The \x02Carnot\x03 cycle", "The Carnot cycle", []domain.UnitSpan{{From: 4, To: 10}}},
		{
			"\x02Heat\x03 and \x02work\x03",
			"Heat and work",
			[]domain.UnitSpan{{From: 0, To: 4}, {From: 9, To: 13}},
		},
		// A run with nothing in it says nothing, and is not one.
		{"\x02\x03Entropy", "Entropy", nil},
		// A character outside the basic plane is two code units to a client,
		// and a run after one begins that much further along.
		{"👋 \x02Entropy\x03", "👋 Entropy", []domain.UnitSpan{{From: 3, To: 10}}},
		{"\x02👋\x03 Entropy", "👋 Entropy", []domain.UnitSpan{{From: 0, To: 2}}},
	} {
		name, at := split(c.marked)
		if name != c.name {
			t.Errorf("split(%q) read %q, want %q", c.marked, name, c.name)
		}
		if len(at) != len(c.at) {
			t.Errorf("split(%q) marked %+v, want %+v", c.marked, at, c.at)
			continue
		}
		for i := range at {
			if at[i] != c.at[i] {
				t.Errorf("split(%q) marked %+v, want %+v", c.marked, at, c.at)
				break
			}
		}
	}
}
