package note_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
)

func TestGetTitleAnswersTheTitleOfTheNoteAtAPath(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})
	title := note.NewGetTitle(c.db.Queries())

	if got := title.Execute(t.Context(), c.vault, "Heat.md"); got == "" {
		t.Error("a note in the index has no title")
	}
	if got := title.Execute(t.Context(), c.vault, "Gone.md"); got != "" {
		t.Errorf("a path naming no note is titled %q", got)
	}
	if got := title.Execute(t.Context(), c.vault, ""); got != "" {
		t.Errorf("no path is titled %q", got)
	}
}

func TestGetTitleAnswersNothingWhereNoIndexIsBound(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})

	if got := note.NewGetTitle(nil).Execute(t.Context(), c.vault, "Heat.md"); got != "" {
		t.Errorf("the note is titled %q", got)
	}
}
