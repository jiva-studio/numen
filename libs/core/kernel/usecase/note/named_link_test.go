package note_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/note/domain"
)

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
