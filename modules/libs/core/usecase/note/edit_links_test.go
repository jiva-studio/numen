package note_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/markdown"
)

func toEntropy(role domain.LinkRole) domain.Link {
	return domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Entropy"}, Role: role}
}

const linkedHeat = "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJRC\nlinks:\n  - to: Entropy\n    role: parent\n    type: series\n---\n# Heat\n"

func TestAddingLinksRefusesAnyThatTheNoteCannotCarry(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		first domain.Link
		more  []domain.Link
	}{
		"the first":  {toEntropy("sibling"), nil},
		"a later":    {toEntropy(domain.RoleParent), []domain.Link{toEntropy(domain.RoleRef), toEntropy("sibling")}},
		"no address": {toEntropy(domain.RoleParent), []domain.Link{{Role: domain.RoleRef}}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})

			_, err := c.newEditLinks().Add(t.Context(), c.vault, "Heat.md", domain.Fingerprint{}, tc.first, tc.more...)

			if err == nil {
				t.Error("a link the note cannot carry was written")
			}
			if body := c.read(t, "Heat.md"); body != "# Heat\n" {
				t.Errorf("the note was written anyway:\n%s", body)
			}
		})
	}
}

func TestCheckingALinkNamesWhatItLacks(t *testing.T) {
	t.Parallel()
	if err := note.CheckWritable(toEntropy(domain.RoleChild)); err != nil {
		t.Errorf("a link with a role and a place failed: %v", err)
	}
	if err := note.CheckWritable(toEntropy("sibling")); err == nil || !strings.Contains(err.Error(), "sibling") {
		t.Errorf("a role the application does not act on was not rejected by name: %v", err)
	}
	if err := note.CheckWritable(domain.Link{Role: domain.RoleParent}); err == nil || !strings.Contains(err.Error(), "somewhere to go") {
		t.Errorf("a link with nowhere to go was not rejected for that: %v", err)
	}
}

// What the caller presents is what the write is held to.
func TestLinkEditsAreHeldToTheFingerprintTheCallerPresents(t *testing.T) {
	t.Parallel()
	edits := map[string]func(c changing, held domain.Fingerprint) error{
		"adding": func(c changing, held domain.Fingerprint) error {
			_, err := c.newEditLinks().Add(t.Context(), c.vault, "Heat.md", held, toEntropy(domain.RoleRef))
			return err
		},
		"pointing": func(c changing, held domain.Fingerprint) error {
			_, err := c.newEditLinks().PointAt(t.Context(), c.vault, "Heat.md", "series",
				domain.Address{Scheme: domain.SchemeName, Value: "Order"}, domain.RoleParent, held)
			return err
		},
		"removing": func(c changing, held domain.Fingerprint) error {
			_, err := c.newEditLinks().Remove(t.Context(), c.vault, "Heat.md",
				domain.Address{Scheme: domain.SchemeName, Value: "Entropy"}, domain.RoleParent, held)
			return err
		},
	}
	for name, edit := range edits {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := openChanging(t, map[string]string{"Heat.md": linkedHeat})
			target := c.openNote(t, "Heat.md").Fingerprint
			theirs := linkedHeat + "\nSaved in the meantime.\n"
			c.overwrite(t, "Heat.md", theirs)

			if err := edit(c, target); !errors.Is(err, port.ErrStale) {
				t.Errorf("want ErrStale, got %v", err)
			}
			if body := c.read(t, "Heat.md"); body != theirs {
				t.Errorf("the edit wrote over a note that had changed:\n%s", body)
			}
		})
	}
}

func TestEditingLinksInABlockSavedOnOneLineChangesNothing(t *testing.T) {
	t.Parallel()
	const inline = "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJRC\nlinks: [Entropy, Order]\n---\n# Heat\n"
	c := openChanging(t, map[string]string{"Heat.md": inline})

	_, added := c.newEditLinks().Add(t.Context(), c.vault, "Heat.md", domain.Fingerprint{}, toEntropy(domain.RoleRef))
	_, removed := c.newEditLinks().Remove(t.Context(), c.vault, "Heat.md",
		domain.Address{Scheme: domain.SchemeName, Value: "Entropy"}, domain.RoleParent, domain.Fingerprint{})

	if !errors.Is(added, markdown.ErrInline) {
		t.Errorf("adding: want ErrInline, got %v", added)
	}
	if !errors.Is(removed, markdown.ErrInline) {
		t.Errorf("removing: want ErrInline, got %v", removed)
	}
	if body := c.read(t, "Heat.md"); body != inline {
		t.Errorf("the note was written anyway:\n%s", body)
	}
}

func TestRemovingALinkThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": linkedHeat})

	_, err := c.newEditLinks().Remove(t.Context(), c.vault, "Heat.md",
		domain.Address{Scheme: domain.SchemeName, Value: "Order"}, domain.RoleParent, domain.Fingerprint{})

	if err == nil || !strings.Contains(err.Error(), "no link to") {
		t.Errorf("want an error naming the missing link, got %v", err)
	}
	if body := c.read(t, "Heat.md"); body != linkedHeat {
		t.Errorf("the note was written anyway:\n%s", body)
	}
}

func TestPointingALinkNeedsATypeAndARoleThatIsKnown(t *testing.T) {
	t.Parallel()
	order := domain.Address{Scheme: domain.SchemeName, Value: "Order"}

	t.Run("no type", func(t *testing.T) {
		t.Parallel()
		c := openChanging(t, map[string]string{"Heat.md": linkedHeat})
		_, err := c.newEditLinks().PointAt(t.Context(), c.vault, "Heat.md", "", order, domain.RoleParent, domain.Fingerprint{})
		if err == nil {
			t.Error("a link pointed at under no type was written")
		}
		if body := c.read(t, "Heat.md"); body != linkedHeat {
			t.Errorf("the note was written anyway:\n%s", body)
		}
	})
	t.Run("a role nobody acts on", func(t *testing.T) {
		t.Parallel()
		c := openChanging(t, map[string]string{"Heat.md": linkedHeat})
		_, err := c.newEditLinks().PointAt(t.Context(), c.vault, "Heat.md", "series", order, "sibling", domain.Fingerprint{})
		if err == nil || !strings.Contains(err.Error(), "sibling") {
			t.Errorf("want an error naming the role, got %v", err)
		}
		if body := c.read(t, "Heat.md"); body != linkedHeat {
			t.Errorf("the note was written anyway:\n%s", body)
		}
	})
	t.Run("a role is not asked for when the link is taken out", func(t *testing.T) {
		t.Parallel()
		c := openChanging(t, map[string]string{"Heat.md": linkedHeat})
		if _, err := c.newEditLinks().PointAt(t.Context(), c.vault, "Heat.md", "series", domain.Address{}, "", domain.Fingerprint{}); err != nil {
			t.Fatalf("taking a link out failed: %v", err)
		}
		if body := c.read(t, "Heat.md"); strings.Contains(body, "Entropy") {
			t.Errorf("the link is still written:\n%s", body)
		}
	})
}
