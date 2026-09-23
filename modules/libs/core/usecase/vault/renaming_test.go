package vault_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// The file tree renames a file, and the note in it is called by the name the
// file now carries.

// title is what the vault shows the note at this path as, asked of the index
// rather than of the file: it is the answer the person sees.
func (f filing) title(t *testing.T, path string) string {
	t.Helper()
	shown, err := f.db.Queries().Notes(t.Context(), f.vault.ID, []string{path})
	if err != nil {
		t.Fatal(err)
	}
	return shown[path].Title
}

// Whichever of the title and the filename names the note is the one brought
// into line, and a note its filename names has nothing to write.
func TestARenamedFileCallsTheNoteByTheNameItNowCarries(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		raw   string
		holds []string
		lacks []string
	}{
		"a title in the frontmatter, with a heading below it left alone": {
			raw:   "---\ntitle: Entropy\n---\n# Entropy\n",
			holds: []string{"title: Disorder", "# Entropy"},
		},
		"a level-one heading, which names nothing and is left standing": {
			raw:   "# Entropy\n\nA measure.\n",
			holds: []string{"# Entropy", "A measure."},
			lacks: []string{"# Disorder", "title:", "id: "},
		},
		"neither, so the file carried the name and nothing was added": {
			raw:   "A measure.\n",
			holds: []string{"A measure."},
			lacks: []string{"title:", "id: ", "# "},
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := openFiling(t, map[string]string{"Entropy.md": c.raw})

			moved, err := f.move().Execute(t.Context(), f.vault, "Entropy.md", "Disorder.md")
			if err != nil {
				t.Fatal(err)
			}
			if !moved.IsLanded {
				t.Fatal("the file did not move")
			}
			body := f.read(t, "Disorder.md")
			for _, kept := range c.holds {
				if !strings.Contains(body, kept) {
					t.Errorf("want %q in\n%s", kept, body)
				}
			}
			for _, missing := range c.lacks {
				if strings.Contains(body, missing) {
					t.Errorf("want no %q in\n%s", missing, body)
				}
			}
			if got := f.title(t, "Disorder.md"); got != "Disorder" {
				t.Errorf("the vault shows it as %q", got)
			}
		})
	}
}

// Where a title and a filename are told apart, a renamed file leaves the note
// as it was written. A note its filename names still changes its name, because
// it carries it nowhere else.
func TestARenamedFileLeavesTheNoteAloneWhereTheTwoAreToldApart(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		raw   string
		shown string
	}{
		"a title in the frontmatter": {
			raw: "---\ntitle: Entropy\n---\nA measure.\n", shown: "Entropy",
		},
		"a level-one heading, which names nothing": {
			raw: "# Entropy\n\nA measure.\n", shown: "Disorder",
		},
		"neither, so the file carries the name": {
			raw: "A measure.\n", shown: "Disorder",
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := openFiling(t, map[string]string{"Entropy.md": c.raw})

			if _, err := f.moveApart().Execute(t.Context(), f.vault, "Entropy.md", "Disorder.md"); err != nil {
				t.Fatal(err)
			}
			if got := f.read(t, "Disorder.md"); got != c.raw {
				t.Errorf("the note was written to:\n%s", got)
			}
			if got := f.title(t, "Disorder.md"); got != c.shown {
				t.Errorf("the vault shows it as %q", got)
			}
		})
	}
}

// A move between folders is not a rename, so the note keeps the name it had
// however the two are held.
func TestAFileFiledUnderAnotherFolderKeepsTheNameItHad(t *testing.T) {
	t.Parallel()
	for name, moving := range map[string]func(filing) vaults.Move{
		"one name":   filing.move,
		"told apart": filing.moveApart,
	} {
		t.Run(name, func(t *testing.T) {
			f := openFiling(t, map[string]string{
				"physics/Entropy.md": "---\ntitle: Entropy\n---\nA measure.\n",
			})

			_, err := moving(f).Execute(t.Context(), f.vault, "physics/Entropy.md", "archive/Disorder.md")
			if err != nil {
				t.Fatal(err)
			}
			if got := f.read(t, "archive/Disorder.md"); !strings.Contains(got, "title: Entropy") {
				t.Errorf("the note was written to:\n%s", got)
			}
			if got := f.title(t, "archive/Disorder.md"); got != "Entropy" {
				t.Errorf("the vault shows it as %q", got)
			}
		})
	}
}

// A folder renamed is not a note renamed, so nothing under it is written to
// however the two are held.
func TestARenamedFolderWritesToNothingUnderIt(t *testing.T) {
	t.Parallel()
	for name, moving := range map[string]func(filing) vaults.Move{
		"one name":   filing.move,
		"told apart": filing.moveApart,
	} {
		t.Run(name, func(t *testing.T) {
			notes := map[string]string{
				"physics/Entropy.md": "---\ntitle: Entropy\n---\nA measure.\n",
				"physics/Heat.md":    "# Heat\n",
			}
			f := openFiling(t, notes)

			if _, err := moving(f).Execute(t.Context(), f.vault, "physics", "chemistry"); err != nil {
				t.Fatal(err)
			}
			for path, raw := range notes {
				at := "chemistry" + strings.TrimPrefix(path, "physics")
				if got := f.read(t, at); got != raw {
					t.Errorf("%s was written to:\n%s", at, got)
				}
			}
		})
	}
}

// A note its filename names carries its name nowhere else, so a renamed file is
// the whole of the rename and the prose is left as it was written.
func TestARenamedFileLeavesANoteCarryingNoTitleAlone(t *testing.T) {
	t.Parallel()
	raw := "# Entropy\n\nA measure.\n"
	f := openFiling(t, map[string]string{"Entropy.md": raw})

	moved, err := f.move().Execute(t.Context(), f.vault, "Entropy.md", "Note #.md")
	if err != nil {
		t.Fatalf("the rename was refused: %v", err)
	}
	if !moved.IsLanded {
		t.Error("the file is at Note #.md and the answer says it did not move")
	}
	if got := f.read(t, "Note #.md"); got != raw {
		t.Errorf("the note was written to:\n%s", got)
	}
}

// A move that landed is settled whatever the note's own name did. Whoever is
// drawing the note at the name it had is reading a name with no file behind it.
func TestAMoveThatLandedIsSettled(t *testing.T) {
	t.Parallel()
	f := openFiling(t, map[string]string{"Entropy.md": "# Entropy\n\nA measure.\n"})

	if _, err := f.move().Execute(t.Context(), f.vault, "Entropy.md", "Note #.md"); err != nil {
		t.Fatal(err)
	}
	if len(*f.went) != 1 || (*f.went)[0].To != "Note #.md" {
		t.Errorf("whoever is drawing it was told %+v", *f.went)
	}
}

// A note whose frontmatter cannot be read is never written, and renaming its
// file is not the moment to repair it.
func TestARenamedFileLeavesANoteWhoseFrontmatterCannotBeReadAlone(t *testing.T) {
	t.Parallel()
	raw := "---\nid: [unterminated\n---\n# Entropy\n"
	f := openFiling(t, map[string]string{"Entropy.md": raw})

	moved, err := f.move().Execute(t.Context(), f.vault, "Entropy.md", "Disorder.md")
	if err != nil {
		t.Fatalf("the rename was refused: %v", err)
	}
	if !moved.IsLanded {
		t.Fatal("the file did not move")
	}
	if got := f.read(t, "Disorder.md"); got != raw {
		t.Errorf("the note was written to:\n%s", got)
	}
}

// Where the two are told apart, a renamed file is not read at all. The note is
// opened only where the setting writes into it.
func TestARenamedFileIsNotReadWhereTheTwoAreToldApart(t *testing.T) {
	t.Parallel()
	f := openFiling(t, map[string]string{"Entropy.md": "---\ntitle: Entropy\n---\nA measure.\n"})
	before := f.readers.reads

	if _, err := f.moveApart().Execute(t.Context(), f.vault, "Entropy.md", "Disorder.md"); err != nil {
		t.Fatal(err)
	}
	if got := f.readers.reads - before; got != 0 {
		t.Errorf("%d files were read", got)
	}
}

// A file filed under another extension is a file of another kind, so it is not
// a note given a different name.
func TestAFileGivenAnotherExtensionIsNotANoteRenamed(t *testing.T) {
	t.Parallel()
	raw := "---\ntitle: Entropy\n---\nA measure.\n"
	f := openFiling(t, map[string]string{"Entropy.md": raw})

	if _, err := f.move().Execute(t.Context(), f.vault, "Entropy.md", "Notes.txt"); err != nil {
		t.Fatal(err)
	}
	if got := f.read(t, "Notes.txt"); got != raw {
		t.Errorf("a file that is no longer a note was written to:\n%s", got)
	}
}

// A file the index holds nothing about is renamed like any other, and there is
// no note in it to call anything.
func TestRenamingAFileTheIndexHoldsNothingAbout(t *testing.T) {
	t.Parallel()
	f := openFiling(t, map[string]string{"Entropy.md": "# Entropy\n"})
	loose := filepath.Join(f.vault.Path, "notes.txt")
	if err := os.WriteFile(loose, []byte("A measure.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	moved, err := f.move().Execute(t.Context(), f.vault, "notes.txt", "other.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !moved.IsLanded {
		t.Fatal("the file did not move")
	}
	if got := f.read(t, "other.txt"); got != "A measure.\n" {
		t.Errorf("the file was written to:\n%s", got)
	}
}

// A name already taken moves nothing, so the note is not left called one thing
// and filed under another.
func TestARenameOntoATakenNameLeavesTheNoteCalledWhatItWas(t *testing.T) {
	t.Parallel()
	raw := "---\ntitle: Entropy\n---\nA measure.\n"
	f := openFiling(t, map[string]string{"Entropy.md": raw, "Disorder.md": "# Disorder\n"})

	moved, err := f.move().Execute(t.Context(), f.vault, "Entropy.md", "Disorder.md")
	if !errors.Is(err, port.ErrOccupied) {
		t.Fatalf("want ErrOccupied, got %v", err)
	}
	if moved.IsLanded {
		t.Error("the move says it landed")
	}
	if got := f.read(t, "Entropy.md"); got != raw {
		t.Errorf("the note was written to:\n%s", got)
	}
	if got := f.read(t, "Disorder.md"); got != "# Disorder\n" {
		t.Errorf("the note already there was written over:\n%s", got)
	}
	if got := f.title(t, "Entropy.md"); got != "Entropy" {
		t.Errorf("the vault shows it as %q", got)
	}
}
