package cards_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
)

// A vault past the bound is refused before the stencil is touched, and the same
// rename with no bound goes through.
func TestRenamingAFieldIsRefusedWhereTheVaultHoldsMoreDecksThanTheBound(t *testing.T) {
	vs := newVaults(t)
	before := read(t, vs.first, "Animal.md")

	_, err := newRenameField(t, vs).WithMostDecks(1).Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if err == nil || !strings.Contains(err.Error(), "there are 2 of them") || !strings.Contains(err.Error(), "more than the 1") {
		t.Fatalf("a vault past the bound was renamed in: %v", err)
	}
	if read(t, vs.first, "Animal.md") != before {
		t.Error("the stencil was written")
	}

	if _, err := newRenameField(t, vs).WithMostDecks(2).Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	}); err != nil {
		t.Errorf("a vault at the bound was refused: %v", err)
	}
}

func TestAStencilIsMadeWithItsFacesLaidDown(t *testing.T) {
	v := newCreateVault(t)
	u := cards.NewCreate(filesystem.VaultWriters{}, nil, time.Now)

	made, err := u.StencilWithFaces(t.Context(), v, cards.New{Title: "Bird", Fields: []string{"Name", "Call"}},
		[]format.FaceTemplate{{Name: "Recognise", Front: "{{Name}}", Back: "{{Call}}"}})
	if err != nil {
		t.Fatal(err)
	}
	body := read(t, v, made.Path)
	if !strings.Contains(body, "## Recognise") || !strings.Contains(body, "{{Call}}") {
		t.Errorf("the faces are not in the stencil:\n%s", body)
	}
}

func TestAStencilWithFacesLargerThanANoteIsRefused(t *testing.T) {
	v := newCreateVault(t)
	u := cards.NewCreate(filesystem.VaultWriters{}, nil, time.Now)

	_, err := u.StencilWithFaces(t.Context(), v, cards.New{Title: "Bird", Fields: []string{"Name"}},
		[]format.FaceTemplate{{Name: "Big", Front: strings.Repeat("x", 1<<20), Back: "{{Name}}"}})
	if err == nil || !strings.Contains(err.Error(), fmt.Sprint("more than this writes at once")) {
		t.Errorf("an oversized stencil was made: %v", err)
	}
}
