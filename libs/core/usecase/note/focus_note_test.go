package note_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// looking is a window that records the places it is requested to show.
type looking struct {
	requested []domain.Place
	fails     error
}

func (l *looking) FocusPlace(_ context.Context, pos domain.Place) error {
	if l.fails != nil {
		return l.fails
	}
	l.requested = append(l.requested, pos)
	return nil
}
func (l *looking) ShowMove(context.Context, domain.Move) error { return nil }
func (l *looking) ShowEdit(context.Context, domain.Edit) error { return nil }

func TestFocusingANoteInsertsItInFrontOfThePerson(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})
	view := &looking{}
	focus := note.NewFocusNote(c.db.Queries(), view)

	ref, err := focus.Execute(t.Context(), c.vault, "Heat.md")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Path != "Heat.md" || len(view.requested) != 1 || view.requested[0].Path != "Heat.md" {
		t.Errorf("focused %+v, window asked %+v", ref, view.requested)
	}
}

func TestFocusingNothingOrANoteThatIsNotThereMovesNobody(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})
	view := &looking{}
	focus := note.NewFocusNote(c.db.Queries(), view)

	if _, err := focus.Execute(t.Context(), c.vault, ""); err == nil {
		t.Error("no path was focused")
	}
	if _, err := focus.Execute(t.Context(), c.vault, "Gone.md"); err == nil {
		t.Error("a note that is not there was focused")
	}
	if len(view.requested) != 0 {
		t.Errorf("the person was moved to %+v", view.requested)
	}

	view.fails = errors.New("no active view")
	if _, err := focus.Execute(t.Context(), c.vault, "Heat.md"); !errors.Is(err, view.fails) {
		t.Errorf("a window that failed answered %v", err)
	}
}
