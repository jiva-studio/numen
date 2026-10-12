package note_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

func TestTitlesAnswerInTheOrderAskedAndNameWhatIsMissing(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n", "Work.md": "# Work\n"})

	got, err := note.NewGetTitles(c.db.Queries()).Execute(t.Context(), c.vault,
		[]string{"Work.md", "Gone.md", "Heat.md", "Lost.md"})
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, ref := range got.Notes {
		paths = append(paths, ref.Path)
	}
	if !slices.Equal(paths, []string{"Work.md", "Heat.md"}) {
		t.Errorf("the notes came back as %v", paths)
	}
	if !slices.Equal(got.Missing, []string{"Gone.md", "Lost.md"}) {
		t.Errorf("the missing came back as %v", got.Missing)
	}
}

func TestANameResolvesToEveryNoteFiledUnderIt(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"Heat.md": "# Heat\n", "deep/Heat.md": "# Heat\n", "Work.md": "# Work\n",
	})
	resolve := note.NewGetNamedPaths(c.db.Queries())

	got, err := resolve.Execute(t.Context(), c.vault, "Heat")
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	if !slices.Equal(got, []string{"Heat.md", "deep/Heat.md"}) {
		t.Errorf("Heat resolved to %v", got)
	}
	if none, _ := resolve.Execute(t.Context(), c.vault, "Nothing"); len(none) != 0 {
		t.Errorf("a name nobody answers to resolved to %v", none)
	}
}

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

func TestRemovingANoteRefusesAFolderAndKeepsTheRestOfTheCall(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"dir/Heat.md": "# Heat\n", "Work.md": "# Work\n"})
	remove := note.NewRemoveNote(c.remove(), filesystem.VaultReaders{})

	if _, err := remove.Execute(t.Context(), c.vault, "dir", false); !errors.Is(err, note.ErrFolder) {
		t.Errorf("a folder was removed: %v", err)
	}
	if !c.exists("dir/Heat.md") {
		t.Error("the folder's note went")
	}

	trashed, err := remove.Execute(t.Context(), c.vault, "Work.md", false)
	if err != nil {
		t.Fatal(err)
	}
	if trashed.Trashed == "" || c.exists("Work.md") {
		t.Errorf("the note was not trashed: %+v", trashed)
	}

	destroyed, err := remove.Execute(t.Context(), c.vault, "dir/Heat.md", true)
	if err != nil {
		t.Fatal(err)
	}
	if destroyed.Trashed != "" || c.exists("dir/Heat.md") {
		t.Errorf("the note was not destroyed: %+v", destroyed)
	}
}

func TestMovingANoteIntoAFolderKeepsItsName(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})

	moved, err := note.NewMoveInto(c.move()).Execute(t.Context(), c.vault, "Heat.md", "physics")
	if err != nil {
		t.Fatal(err)
	}
	if moved.To != "physics/Heat.md" || !c.exists("physics/Heat.md") || c.exists("Heat.md") {
		t.Errorf("the note moved as %+v", moved)
	}
}
