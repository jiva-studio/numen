package note_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

func (c changing) rename() note.Rename {
	return note.Rename{
		Readers: filesystem.Readers{}, Writers: filesystem.Writers{},
		Links: c.db.Links(), Index: c.index,
	}
}

// title is what the vault shows the note at this path as, asked of the index
// rather than of the file: it is the answer the person sees.
func (c changing) title(t *testing.T, path string) string {
	t.Helper()
	shown, err := c.db.Queries().Notes(t.Context(), c.vault.ID, []string{path})
	if err != nil {
		t.Fatal(err)
	}
	return shown[path].Title
}

// Whichever of the title, the heading and the filename names the note is the
// one brought into line, and the ones below it are left as they were written.
func TestRenamingWritesWhateverNamesTheNote(t *testing.T) {
	for name, c := range map[string]struct {
		raw   string
		title string
		path  string
		by    string
		holds []string
		lacks []string
	}{
		"a title in the frontmatter, with a heading below it left alone": {
			raw:   "---\ntitle: Old\n---\n# Old\n",
			title: "Entropy",
			path:  "Entropy.md",
			by:    "frontmatter",
			holds: []string{"title: Entropy", "# Old"},
		},
		"a level-one heading": {
			raw:   "# Old\n\nA measure.\n",
			title: "Entropy",
			path:  "Entropy.md",
			by:    "heading",
			holds: []string{"# Entropy", "A measure.", "id: "},
			lacks: []string{"# Old", "title:"},
		},
		"neither, and the filename cannot carry the title": {
			raw:   "A measure.\n",
			title: "TCP/IP",
			path:  "TCP-IP.md",
			by:    "heading",
			holds: []string{"# TCP/IP", "A measure.", "id: "},
			lacks: []string{"title:"},
		},
		"neither, and the filename says it": {
			raw:   "A measure.\n",
			title: "Entropy",
			path:  "Entropy.md",
			by:    "filename",
			holds: []string{"A measure."},
			lacks: []string{"id: ", "# Entropy", "title:"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			v := changeable(t, map[string]string{"Old.md": c.raw})

			renamed, err := v.rename().Execute(t.Context(), v.vault, "Old.md", c.title)
			if err != nil {
				t.Fatal(err)
			}
			if renamed.Path != c.path {
				t.Errorf("want %s, got %s", c.path, renamed.Path)
			}
			if renamed.By != c.by {
				t.Errorf("want named by %s, got %s", c.by, renamed.By)
			}
			if renamed.Title != c.title {
				t.Errorf("want %q, got %q", c.title, renamed.Title)
			}
			if renamed.Moved == nil || renamed.Moved.From != "Old.md" {
				t.Errorf("want the file's move reported, got %+v", renamed.Moved)
			}

			body := v.read(t, renamed.Path)
			for _, kept := range c.holds {
				if !strings.Contains(body, kept) {
					t.Errorf("want %q in\n%s", kept, body)
				}
			}
			for _, gone := range c.lacks {
				if strings.Contains(body, gone) {
					t.Errorf("want no %q in\n%s", gone, body)
				}
			}
			if got := v.title(t, renamed.Path); got != c.title {
				t.Errorf("the vault shows it as %q", got)
			}
		})
	}
}

// A note whose filename is the whole of its naming is moved and not edited, so
// it comes out of a rename with the bytes it went in with.
func TestRenamingByTheFilenameAloneLeavesTheBytesAlone(t *testing.T) {
	c := changeable(t, map[string]string{"Old.md": "A measure.\n"})

	renamed, err := c.rename().Execute(t.Context(), c.vault, "Old.md", "Entropy")
	if err != nil {
		t.Fatal(err)
	}
	if got := c.read(t, renamed.Path); got != "A measure.\n" {
		t.Errorf("want the note untouched, got %q", got)
	}
}

// The filename is already what the title reduces to, so there is nothing for
// the file to do and nothing to report about it.
func TestRenamingCanLeaveTheFileWhereItIs(t *testing.T) {
	c := changeable(t, map[string]string{"Entropy.md": "# Old\n"})

	renamed, err := c.rename().Execute(t.Context(), c.vault, "Entropy.md", "Entropy")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Moved != nil {
		t.Errorf("the file did not move: %+v", renamed.Moved)
	}
	if renamed.Path != "Entropy.md" {
		t.Errorf("want Entropy.md, got %s", renamed.Path)
	}
	if body := c.read(t, "Entropy.md"); !strings.Contains(body, "# Entropy") {
		t.Errorf("the heading was not rewritten:\n%s", body)
	}
}

// The note is brought into line before the file is, so a refused move leaves a
// note that says what it is called under a filename that does not.
func TestRenamingRefusesToLandOnAnExistingNote(t *testing.T) {
	c := changeable(t, map[string]string{
		"Old.md":     "# Old\n",
		"Entropy.md": "# Entropy\n",
	})

	renamed, err := c.rename().Execute(t.Context(), c.vault, "Old.md", "Entropy")
	if !errors.Is(err, port.ErrOccupied) {
		t.Fatalf("want ErrOccupied, got %v", err)
	}
	if renamed.Path != "Old.md" {
		t.Errorf("want the note where it still is, got %s", renamed.Path)
	}
	if body := c.read(t, "Old.md"); !strings.Contains(body, "# Entropy") {
		t.Errorf("the title was not written:\n%s", body)
	}
	if body := c.read(t, "Entropy.md"); body != "# Entropy\n" {
		t.Errorf("the note already there was written over:\n%s", body)
	}
}

func TestRenamingRefusesATitleThatCannotBeAFilename(t *testing.T) {
	for name, title := range map[string]string{
		"nothing at all": "",
		"only spaces":    "   ",
		"only dots":      "...",
		"only controls":  "\x00\x01",
	} {
		t.Run(name, func(t *testing.T) {
			c := changeable(t, map[string]string{"Old.md": "# Old\n"})

			if _, err := c.rename().Execute(t.Context(), c.vault, "Old.md", title); err == nil {
				t.Fatal("want a refusal")
			}
			if body := c.read(t, "Old.md"); body != "# Old\n" {
				t.Errorf("the refused rename wrote to the note:\n%s", body)
			}
		})
	}
}

// A title longer than a filename will take is cut to make the name, and the
// whole of it goes into the note, where the order of resolution finds it.
func TestALongTitleIsCutFromTheNameAndKeptWhole(t *testing.T) {
	c := changeable(t, map[string]string{"Old.md": "A measure.\n"})
	title := strings.TrimSpace(strings.Repeat("disorder ", 20))

	renamed, err := c.rename().Execute(t.Context(), c.vault, "Old.md", title)
	if err != nil {
		t.Fatal(err)
	}
	if len(renamed.Path) >= len(title) {
		t.Errorf("the name was not cut: %s", renamed.Path)
	}
	if renamed.By != "heading" {
		t.Errorf("want named by heading, got %s", renamed.By)
	}
	if body := c.read(t, renamed.Path); !strings.Contains(body, "# "+title) {
		t.Errorf("the whole title is not in the note:\n%s", body)
	}
	if got := c.title(t, renamed.Path); got != title {
		t.Errorf("the vault shows it as %q", got)
	}
}

// The file half of a rename is a move, so a link that stopped resolving is
// repaired by the machinery a move already has.
func TestRenamingRepairsALinkThatStoppedResolving(t *testing.T) {
	c := changeable(t, map[string]string{
		"physics/Entropy.md": "# Entropy\n",
		"physics/Heat.md":    "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n---\n# Heat\n",
	})

	renamed, err := c.rename().Execute(t.Context(), c.vault, "physics/Entropy.md", "Thermodynamics")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Path != "physics/Thermodynamics.md" {
		t.Fatalf("want the note beside the one that points at it, got %s", renamed.Path)
	}
	if renamed.Moved == nil || len(renamed.Moved.Repaired) != 1 ||
		renamed.Moved.Repaired[0] != "physics/Heat.md" {
		t.Fatalf("want the note whose link broke, got %+v", renamed.Moved)
	}

	found := links(t, c.db, c.vault, "physics/Heat.md")
	if len(found.Links) != 1 || found.Links[0].To != renamed.Path {
		t.Errorf("the repaired link does not reach the note: %+v", found.Links)
	}
}
