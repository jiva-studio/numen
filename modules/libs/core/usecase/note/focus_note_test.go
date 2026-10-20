package note_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// looking is a window that records what is put in front of the person.
type looking struct {
	asked []domain.Place
	fails error
}

func (l *looking) FocusPlace(_ context.Context, at domain.Place) error {
	if l.fails != nil {
		return l.fails
	}
	l.asked = append(l.asked, at)
	return nil
}
func (l *looking) ShowMove(context.Context, domain.Move) error { return nil }
func (l *looking) ShowEdit(context.Context, domain.Edit) error { return nil }

func TestFocusingANotePutsItInFrontOfThePerson(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})
	view := &looking{}
	focus := note.NewFocusNote(c.db.Queries(), view)

	ref, err := focus.Execute(t.Context(), c.vault, "Heat.md")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Path != "Heat.md" || len(view.asked) != 1 || view.asked[0].Path != "Heat.md" {
		t.Errorf("focused %+v, window asked %+v", ref, view.asked)
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
	if len(view.asked) != 0 {
		t.Errorf("the person was moved to %+v", view.asked)
	}

	view.fails = errors.New("nobody is looking")
	if _, err := focus.Execute(t.Context(), c.vault, "Heat.md"); !errors.Is(err, view.fails) {
		t.Errorf("a window that failed answered %v", err)
	}
}
