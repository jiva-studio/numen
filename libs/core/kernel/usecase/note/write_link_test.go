package note_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// windowGate is the writes a window waits for; a closing one admits none.
type windowGate struct{ isClosing bool }

func (g windowGate) BeginWrite() bool { return !g.isClosing }
func (g windowGate) FinishWrite()     {}

func TestWriteLinkNamesTheTargetByItsName(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Heat.md": "# Heat\n", "physics/Entropy.md": "# Entropy\n"})
	write := note.NewWriteLink(c.newEditLinks(), c.db.NoteQueries(), windowGate{})

	err := write.Execute(t.Context(), c.vault, "Heat.md", note.NamedLink{To: "physics/Entropy.md", Role: domain.RoleParent})

	if err != nil {
		t.Fatal(err)
	}
	if body := c.read(t, "Heat.md"); !strings.Contains(body, "to: Entropy") {
		t.Errorf("the link does not go to the name of the note:\n%s", body)
	}
}

func TestWriteLinkRefusesALinkWithNowhereToGo(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Heat.md": "# Heat\n"})
	write := note.NewWriteLink(c.newEditLinks(), c.db.NoteQueries(), windowGate{})

	err := write.Execute(t.Context(), c.vault, "Heat.md", note.NamedLink{Role: domain.RoleParent})

	if !errors.Is(err, note.ErrInvalidLink) {
		t.Errorf("a link with no note to go to came back %v", err)
	}
	if body := c.read(t, "Heat.md"); body != "# Heat\n" {
		t.Errorf("the note was written anyway:\n%s", body)
	}
}

func TestWriteLinkWritesNothingWhileTheWindowCloses(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Heat.md": "# Heat\n", "Entropy.md": "# Entropy\n"})
	write := note.NewWriteLink(c.newEditLinks(), c.db.NoteQueries(), windowGate{isClosing: true})

	err := write.Execute(t.Context(), c.vault, "Heat.md", note.NamedLink{To: "Entropy.md", Role: domain.RoleParent})

	if !errors.Is(err, windowdomain.ErrClosing) {
		t.Errorf("a write while the window closes came back %v", err)
	}
	if body := c.read(t, "Heat.md"); body != "# Heat\n" {
		t.Errorf("the note was written anyway:\n%s", body)
	}
}

func TestCreateWritesTheNamedLinksIntoTheNewNote(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"physics/Heat.md": "# Heat\n"})
	creating := c.create()

	made, err := creating.Execute(t.Context(), c.vault, note.NewNote{
		Title: "Order", Named: []note.NamedLink{{To: "physics/Heat.md", Role: domain.RoleParent}},
	})

	if err != nil {
		t.Fatal(err)
	}
	if body := c.read(t, made.Path); !strings.Contains(body, "to: Heat") {
		t.Errorf("the new note does not link to the name of the note:\n%s", body)
	}
}

func TestCreateMakesNoFileForALinkWithNowhereToGo(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Heat.md": "# Heat\n"})
	creating := c.create()

	_, err := creating.Execute(t.Context(), c.vault, note.NewNote{
		Title: "Order", Named: []note.NamedLink{{Role: domain.RoleParent}},
	})

	if !errors.Is(err, note.ErrInvalidLink) {
		t.Errorf("a link with no note to go to came back %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(c.vault.Path, "Order.md")); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("a note was made anyway: %v", statErr)
	}
}
