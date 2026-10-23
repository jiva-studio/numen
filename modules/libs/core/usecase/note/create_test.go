package note_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func TestATitleNoFileCanBeNamedAfterMakesNothing(t *testing.T) {
	t.Parallel()
	c := openChanging(t, nil)

	for _, title := range []string{"", "  ", "two\nlines"} {
		if _, err := c.create().Execute(t.Context(), c.vault, note.NewNote{Title: title}); !errors.Is(err, domain.ErrUnnameable) {
			t.Errorf("title %q: want ErrUnnameable, got %v", title, err)
		}
	}
}

// A link the note cannot carry leaves no file, wherever in the list it stands.
func TestALinkTheNoteCannotCarryPreservesNoFile(t *testing.T) {
	t.Parallel()
	good := domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Entropy"}, Role: domain.RoleParent}
	bad := domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Order"}, Role: "sibling"}

	for name, links := range map[string][]domain.Link{
		"the first":  {bad, good},
		"the second": {good, bad},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := openChanging(t, nil)

			if _, err := c.create().Execute(t.Context(), c.vault, note.NewNote{Title: "Heat", Links: links}); err == nil {
				t.Error("a link with no role the application acts on was written")
			}
			if c.exists("Heat.md") {
				t.Error("a rejected link left a note behind")
			}
		})
	}
}

func TestANoteIsCreatedWithTheLinksItWasGiven(t *testing.T) {
	t.Parallel()
	link := func(to string) domain.Link {
		return domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: to}, Role: domain.RoleParent}
	}

	for name, links := range map[string][]domain.Link{
		"one":  {link("Entropy")},
		"two":  {link("Entropy"), link("Order")},
		"none": nil,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := openChanging(t, nil)

			created, err := c.create().Execute(t.Context(), c.vault, note.NewNote{Title: "Heat", Links: links})
			if err != nil {
				t.Fatal(err)
			}

			body := c.read(t, created.Path)
			for _, l := range links {
				if !strings.Contains(body, "to: "+l.Target.Value) {
					t.Errorf("the link to %s is not written:\n%s", l.Target.Value, body)
				}
			}
			if len(links) == 0 && strings.Contains(body, "links:") {
				t.Errorf("a link was written that nobody gave:\n%s", body)
			}
		})
	}
}

// A filename that carries the whole title names the note, and the frontmatter
// holds no `title` key.
func TestATitleTheFilenameCarriesWritesNoTitleKey(t *testing.T) {
	t.Parallel()
	c := openChanging(t, nil)

	created, err := c.create().Execute(t.Context(), c.vault, note.NewNote{Title: "Entropy"})
	if err != nil {
		t.Fatal(err)
	}

	if body := c.read(t, created.Path); strings.Contains(body, "title:") {
		t.Errorf("the filename already carries the title:\n%s", body)
	}
}

func TestCreatingNeedsNoIndexToLeaveTheNoteSaved(t *testing.T) {
	t.Parallel()
	c := openChanging(t, nil)
	create := note.NewCreate(vaultwriter.VaultWriters{}, c.db.Queries(), nil, time.Now)

	created, err := create.Execute(t.Context(), c.vault, note.NewNote{Title: "Heat"})
	if err != nil {
		t.Fatalf("a create with no index to level failed: %v", err)
	}

	if !c.exists(created.Path) {
		t.Error("the note is not on disk")
	}
}

func TestCreatingInAVaultThatWillNotOpenForWritingMakesNothing(t *testing.T) {
	t.Parallel()
	c := openChanging(t, nil)
	writers := brokenWriters{VaultWriters: vaultwriter.VaultWriters{}, open: errDiskGone}
	create := note.NewCreate(writers, c.db.Queries(), c.index, time.Now)

	created, err := create.Execute(t.Context(), c.vault, note.NewNote{Title: "Heat"})

	if !errors.Is(err, errDiskGone) {
		t.Errorf("want the error the vault gave, got %v", err)
	}
	if created.Path != "" {
		t.Errorf("a note that was never made was reported as %+v", created)
	}
}

// Once the note is on disk everything after it answers with where it is.
func TestANoteMadeBeforeTheIndexFailsIsStillReported(t *testing.T) {
	t.Parallel()
	c := openChanging(t, nil)
	create := c.create().WithIndex(func(context.Context, domain.Vault, []string) error { return errDiskGone })

	created, err := create.Execute(t.Context(), c.vault, note.NewNote{Title: "Heat"})

	if !errors.Is(err, note.ErrUnlevelled) || !errors.Is(err, errDiskGone) {
		t.Errorf("want the index failure named as one, got %v", err)
	}
	if created.Path != "Heat.md" || created.ID == "" || created.Title != "Heat" {
		t.Errorf("the note that exists was reported as %+v", created)
	}
	if !c.exists("Heat.md") {
		t.Error("the note is not on disk")
	}
}

func TestANoteMadeBeforeTheVaultCanBeAskedItsNamesIsStillReported(t *testing.T) {
	t.Parallel()
	c := openChanging(t, nil)
	create := note.NewCreate(vaultwriter.VaultWriters{}, unreachable{}, c.index, time.Now)

	created, err := create.Execute(t.Context(), c.vault, note.NewNote{Title: "Heat"})

	if err == nil {
		t.Error("a vault that could not be asked was not reported")
	}
	if created.Path != "Heat.md" || created.ID == "" {
		t.Errorf("the note that exists was reported as %+v", created)
	}
}
