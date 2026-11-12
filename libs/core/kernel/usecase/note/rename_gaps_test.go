package note_test

import (
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// A rename that cannot be applied reports nothing for a name the note never
// took.
func TestARenameOfANoteThatIsNotThereAnswersWithNothing(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Old.md": "# Old\n"})

	got, err := c.rename().Execute(t.Context(), c.vault, "Missing.md", "Entropy")
	if !errors.Is(err, edit.ErrNoNote) {
		t.Fatalf("want ErrNoNote, got %v", err)
	}
	if got != (note.RenameResult{}) {
		t.Errorf("a rename that failed answered %+v", got)
	}
}

// A file renamed by the user is synchronized to the title when the two names
// are kept as one.
func TestAFilenameIsSavedIntoTheTitleWhereTheTwoAreCachedAsOne(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Entropy.md": "---\ntitle: Old\n---\nA measure.\n"})

	if err := c.move().WriteFilenameAsTitle(t.Context(), c.vault, "Entropy.md"); err != nil {
		t.Fatal(err)
	}
	if body := c.read(t, "Entropy.md"); !strings.Contains(body, "title: Entropy\n") || strings.Contains(body, "Old") {
		t.Errorf("the filename was not written as the title:\n%s", body)
	}
}

// When the two names are reported separately, the note is not opened at all.
func TestAFilenameIsLeftOutOfTheNoteWhereTheTwoAreReportedApart(t *testing.T) {
	t.Parallel()
	raw := "---\ntitle: Old\n---\nA measure.\n"
	c := openChangeFixture(t, map[string]string{"Entropy.md": raw})
	apart := note.NewMove(c.moveApartDeps())

	if err := apart.WriteFilenameAsTitle(t.Context(), c.vault, "Entropy.md"); err != nil {
		t.Fatal(err)
	}
	if body := c.read(t, "Entropy.md"); body != raw {
		t.Errorf("the note changed:\n%s", body)
	}

	// A note that is not there is not looked for either.
	if err := apart.WriteFilenameAsTitle(t.Context(), c.vault, "Missing.md"); err != nil {
		t.Errorf("a vault that writes nothing into notes was asked for one: %v", err)
	}
}

// A note its filename names carries its name nowhere else, and nothing is added
// to it, not even the identifier an edit would stamp.
func TestANoteWithNoTitleIsLeftAloneByItsFilename(t *testing.T) {
	t.Parallel()
	raw := "# Entropy\n\nA measure.\n"
	c := openChangeFixture(t, map[string]string{"Entropy.md": raw})

	if err := c.move().WriteFilenameAsTitle(t.Context(), c.vault, "Entropy.md"); err != nil {
		t.Fatal(err)
	}
	if body := c.read(t, "Entropy.md"); body != raw {
		t.Errorf("a note with no title was written:\n%s", body)
	}
}

func TestAFilenameCannotBeSavedIntoANoteThatIsNotThere(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Old.md": "# Old\n"})

	err := c.move().WriteFilenameAsTitle(t.Context(), c.vault, "Missing.md")
	if !errors.Is(err, edit.ErrNoNote) {
		t.Errorf("want ErrNoNote, got %v", err)
	}
}

func TestARenameIsBuiltOverTheMoveItWasGiven(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Old.md": "# Old\n"})

	renaming := note.NewRename(c.moveDeps(), c.move(), testsupport.OpenGate{})
	if _, err := renaming.Execute(t.Context(), c.vault, "Old.md", "Entropy"); err != nil {
		t.Fatal(err)
	}
	if body := c.read(t, "Entropy.md"); body != "# Old\n" {
		t.Errorf("the note did not move:\n%s", body)
	}
}
