package domain_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestGetNameAfterMove(t *testing.T) {
	for _, tc := range []struct {
		label       string
		from, to    string
		title       string
		isNamed     bool
		wantDisplay string
		wantRenamed bool
	}{
		{"same file name in another folder", "a/Tides.md", "b/Tides.md", "Tides", true, "", false},
		{"unnamed note takes the new file name", "Tides.md", "Currents.md", "", false, "Currents", true},
		{"named note keeps its title when the title gives the new file name", "Tides.md", "Currents.md", "Currents", true, "Currents", true},
		{"named note whose title does not give the new name is left alone", "Tides.md", "Currents.md", "Ocean: Tides", true, "", false},
		{"named note matches the new name after folding", "Tides.md", "ocean tides.md", "Ocean Tides", true, "Ocean Tides", true},
		{"unnamed note moved into a folder with a new name", "Tides.md", "sea/Waves.md", "", false, "Waves", true},
	} {
		t.Run(tc.label, func(t *testing.T) {
			display, isRenamed := domain.GetNameAfterMove(tc.from, tc.to, tc.title, tc.isNamed)
			if display != tc.wantDisplay || isRenamed != tc.wantRenamed {
				t.Errorf("got (%q, %v), want (%q, %v)", display, isRenamed, tc.wantDisplay, tc.wantRenamed)
			}
		})
	}
}
