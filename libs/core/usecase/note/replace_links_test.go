package note_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func (c changing) newReplaceLinks() note.ReplaceLinks {
	return note.NewReplaceLinks(c.newEditLinks(), note.NewGetLinkAddress(nil))
}

const (
	plainHeat    = "# Heat\n"
	plainEntropy = "# Entropy\n"
)

func TestReplacingLinksPreservesEachNoteContainingTheLinkGivenForIt(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": plainHeat, "Entropy.md": plainEntropy})

	err := c.newReplaceLinks().Execute(t.Context(), c.vault, "Heat.md", "Entropy.md", []note.PairLink{
		{From: "Heat.md", To: "Entropy.md", Role: domain.RoleParent, Label: "measured by"},
		{From: "Entropy.md", To: "Heat.md", Role: domain.RoleChild, Label: "measures"},
	})

	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	heat, entropy := c.read(t, "Heat.md"), c.read(t, "Entropy.md")
	for _, want := range []string{"to: Entropy", "role: parent", "measured by"} {
		if !strings.Contains(heat, want) {
			t.Errorf("Heat is missing %q:\n%s", want, heat)
		}
	}
	for _, want := range []string{"to: Heat", "role: child", "measures"} {
		if !strings.Contains(entropy, want) {
			t.Errorf("Entropy is missing %q:\n%s", want, entropy)
		}
	}
}

func TestReplacingLinksWithFewerRemovesTheOnesLeftOut(t *testing.T) {
	t.Parallel()
	both := "---\nlinks:\n  - to: %s\n    role: jump\n---\n"
	c := openChanging(t, map[string]string{
		"Heat.md":    strings.Replace(both, "%s", "Entropy", 1) + plainHeat,
		"Entropy.md": strings.Replace(both, "%s", "Heat", 1) + plainEntropy,
	})

	err := c.newReplaceLinks().Execute(t.Context(), c.vault, "Heat.md", "Entropy.md", []note.PairLink{
		{From: "Entropy.md", To: "Heat.md", Role: domain.RoleRef, Label: "only this"},
	})

	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	if heat := c.read(t, "Heat.md"); strings.Contains(heat, "Entropy") {
		t.Errorf("Heat still names Entropy:\n%s", heat)
	}
	if entropy := c.read(t, "Entropy.md"); !strings.Contains(entropy, "role: ref") ||
		strings.Contains(entropy, "role: jump") {
		t.Errorf("Entropy does not hold the one link given:\n%s", entropy)
	}
}

func TestReplacingLinksWithNoneRemovesBoth(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"Heat.md":    "---\nlinks:\n  - to: Entropy\n    role: jump\n---\n" + plainHeat,
		"Entropy.md": plainEntropy,
	})

	if err := c.newReplaceLinks().Execute(t.Context(), c.vault, "Heat.md", "Entropy.md", nil); err != nil {
		t.Fatalf("replace: %v", err)
	}
	if heat := c.read(t, "Heat.md"); strings.Contains(heat, "Entropy") {
		t.Errorf("Heat still names Entropy:\n%s", heat)
	}
}

func TestReplacingLinksWritesNothingWhenOneLinkIsRefused(t *testing.T) {
	t.Parallel()
	good := note.PairLink{From: "Heat.md", To: "Entropy.md", Role: domain.RoleParent}
	for name, bad := range map[string]note.PairLink{
		"another note":  {From: "Order.md", To: "Entropy.md", Role: domain.RoleChild},
		"the same end":  {From: "Heat.md", To: "Entropy.md", Role: domain.RoleJump},
		"unknown role":  {From: "Entropy.md", To: "Heat.md", Role: "sibling"},
		"to itself":     {From: "Heat.md", To: "Heat.md", Role: domain.RoleJump},
		"into the void": {From: "Entropy.md", To: "", Role: domain.RoleJump},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := openChanging(t, map[string]string{"Heat.md": plainHeat, "Entropy.md": plainEntropy})

			err := c.newReplaceLinks().Execute(
				t.Context(), c.vault, "Heat.md", "Entropy.md", []note.PairLink{good, bad})

			if err == nil {
				t.Fatal("a set of links with a rejected one was written")
			}
			if heat := c.read(t, "Heat.md"); heat != plainHeat {
				t.Errorf("Heat was written before the rejection:\n%s", heat)
			}
		})
	}
}

func TestReplacingLinksReportsAMissingNote(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": plainHeat})

	err := c.newReplaceLinks().Execute(t.Context(), c.vault, "Heat.md", "Gone.md", []note.PairLink{
		{From: "Gone.md", To: "Heat.md", Role: domain.RoleJump},
	})

	if !errors.Is(err, note.ErrNoNote) {
		t.Errorf("want ErrNoNote, got %v", err)
	}
}
