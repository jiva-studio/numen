package note_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
)

var day = time.Date(2026, time.November, 23, 23, 30, 0, 0, time.UTC)

func TestGetConflictCopyPath_NamesTheCopyAfterTheNoteAndTheDay(t *testing.T) {
	got := note.GetConflictCopyPath("Shopping list.md", day, 1)

	if want := "Shopping list (conflict 2026-11-23).md"; got != want {
		t.Errorf("copy path = %q, want %q", got, want)
	}
}

func TestGetConflictCopyPath_KeepsTheFolderOfTheNote(t *testing.T) {
	got := note.GetConflictCopyPath("Food/Daily/Monday.md", day, 1)

	if want := "Food/Daily/Monday (conflict 2026-11-23).md"; got != want {
		t.Errorf("copy path = %q, want %q", got, want)
	}
}

func TestGetConflictCopyPath_AddsTheNumberAfterTheDayFromTheSecondName(t *testing.T) {
	cases := map[int]string{
		2: "Shopping list (conflict 2026-11-23 2).md",
		3: "Shopping list (conflict 2026-11-23 3).md",
	}
	for number, want := range cases {
		if got := note.GetConflictCopyPath("Shopping list.md", day, number); got != want {
			t.Errorf("copy path %d = %q, want %q", number, got, want)
		}
	}
}

func TestGetConflictCopyPath_KeepsTheExtensionInTheCaseOfTheNote(t *testing.T) {
	got := note.GetConflictCopyPath("Shopping list.MD", day, 1)

	if want := "Shopping list (conflict 2026-11-23).md"; got != want {
		t.Errorf("copy path = %q, want the markdown extension in lower case", got)
	}
}

func TestGetConflictCopyPath_ReturnsNormalFormC(t *testing.T) {
	decomposed := "Café.md"

	got := note.GetConflictCopyPath(decomposed, day, 1)

	if want := "Café (conflict 2026-11-23).md"; got != want {
		t.Errorf("copy path = %q, want the composed form %q", got, want)
	}
}

func TestGetConflictCopyPath_UsesTheDayOfTheTimeItIsGiven(t *testing.T) {
	zone := time.FixedZone("east", 5*60*60)
	late := time.Date(2026, time.November, 23, 22, 0, 0, 0, zone)

	got := note.GetConflictCopyPath("Shopping list.md", late, 1)

	if want := "Shopping list (conflict 2026-11-23).md"; got != want {
		t.Errorf("copy path = %q, want the day in the zone of the time", got)
	}
}

func TestIsCaseClash(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"names that differ in case", "Papers/Report.pdf", "Papers/report.pdf", true},
		{"names that differ in composition", "Café.md", "Café.md", false},
		{"names that differ in case and composition", "CafÉ.md", "Café.md", true},
		{"equal names", "Papers/Report.pdf", "Papers/Report.pdf", false},
		{"other names", "Papers/Report.pdf", "Papers/Plan.pdf", false},
		{"names of other folders", "Papers/Report.pdf", "papers2/Report.pdf", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := note.IsCaseClash(c.a, c.b); got != c.want {
				t.Errorf("IsCaseClash(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}
