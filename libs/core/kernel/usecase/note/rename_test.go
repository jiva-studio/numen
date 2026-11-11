package note_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/naming"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
)

func (c changing) rename() note.Rename {
	return note.NewRename(c.moveDeps(), c.move(), note.OpenGate{})
}

func (c changing) moveApartDeps() note.MoveDeps {
	deps := c.moveDeps()
	deps.Sync = func() naming.SyncTitleAndFilename { return false }
	return deps
}

func (c changing) renameApart() note.Rename {
	deps := c.moveApartDeps()
	return note.NewRename(deps, note.NewMove(deps), note.OpenGate{})
}

func (c changing) title(t *testing.T, path string) string {
	t.Helper()
	displayed, err := c.db.NoteQueries().GetNotes(t.Context(), c.vault.ID, []string{path})
	if err != nil {
		t.Fatal(err)
	}
	return displayed[path].Title
}

// The rename updates whichever of the title and the filename names the note,
// and leaves the body text unchanged.
func TestRenamingWritesWhateverNamesTheNote(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		raw   string
		title string
		path  string
		delta naming.Source
		holds []string
		lacks []string
	}{
		"a title in the frontmatter, with a heading below it left alone": {
			raw:   "---\ntitle: Old\n---\n# Old\n",
			title: "Entropy",
			path:  "Entropy.md",
			delta: naming.ByFrontmatter,
			holds: []string{"title: Entropy", "# Old"},
		},
		"a level-one heading, which names nothing and is left standing": {
			raw:   "# Old\n\nA measure.\n",
			title: "Entropy",
			path:  "Entropy.md",
			delta: naming.ByFilename,
			holds: []string{"# Old", "A measure."},
			lacks: []string{"title:", "id: "},
		},
		"no key, and the filename cannot carry the title": {
			raw:   "A measure.\n",
			title: "TCP/IP",
			path:  "TCP-IP.md",
			delta: naming.ByFrontmatter,
			holds: []string{"title: TCP/IP", "A measure.", "id: "},
			lacks: []string{"# TCP/IP"},
		},
		"no key, and the filename determines the title": {
			raw:   "A measure.\n",
			title: "Entropy",
			path:  "Entropy.md",
			delta: naming.ByFilename,
			holds: []string{"A measure."},
			lacks: []string{"id: ", "# Entropy", "title:"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			v := openChanging(t, map[string]string{"Old.md": c.raw})

			renamed, err := v.rename().Execute(t.Context(), v.vault, "Old.md", c.title)
			if err != nil {
				t.Fatal(err)
			}
			if renamed.Path != c.path {
				t.Errorf("want %s, got %s", c.path, renamed.Path)
			}
			if renamed.By != c.delta {
				t.Errorf("want named by %s, got %s", c.delta, renamed.By)
			}
			if renamed.Title != c.title {
				t.Errorf("want %q, got %q", c.title, renamed.Title)
			}
			if renamed.Moved == nil || renamed.Moved.From != "Old.md" {
				t.Errorf("want the file's move reported, got %+v", renamed.Moved)
			}

			body := v.read(t, renamed.Path)
			for _, holdItem := range c.holds {
				if !strings.Contains(body, holdItem) {
					t.Errorf("want %q in\n%s", holdItem, body)
				}
			}
			for _, removed := range c.lacks {
				if strings.Contains(body, removed) {
					t.Errorf("want no %q in\n%s", removed, body)
				}
			}
			if got := v.title(t, renamed.Path); got != c.title {
				t.Errorf("the vault shows it as %q", got)
			}
		})
	}
}

// The vault shows the note under the title given to the rename, whichever of
// the title and the filename holds it. A title that the filename expresses as
// a different name is a defect.
func TestTheVaultShowsTheTitleTheRenameWasGiven(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		raw   string
		title string
	}{
		"a hash inside the title, on a note with no key": {
			raw:   "# Old\n\nA measure.\n",
			title: "Issue #42",
		},
		"a hash the frontmatter carries, on a note carrying the key": {
			raw:   "---\ntitle: Old\n---\nA measure.\n",
			title: "C# and F#",
		},
		"a title in double brackets": {
			raw:   "A measure.\n",
			title: "Notes [[draft]]",
		},
		"a title that is a path": {
			raw:   "A measure.\n",
			title: "TCP/IP",
		},
		"a title a filename takes whole": {
			raw:   "A measure.\n",
			title: "Thermodynamics",
		},
		"a title carrying a pipe": {
			raw:   "A measure.\n",
			title: "Either|Or",
		},
	} {
		t.Run(name, func(t *testing.T) {
			v := openChanging(t, map[string]string{"Old.md": c.raw})

			renamed, err := v.rename().Execute(t.Context(), v.vault, "Old.md", c.title)
			if err != nil {
				t.Fatalf("the rename failed: %v", err)
			}
			if renamed.Title != c.title {
				t.Errorf("the answer says the note is called %q", renamed.Title)
			}
			if got := v.title(t, renamed.Path); got != c.title {
				t.Errorf("the vault shows the note as %q, and it was named %q", got, c.title)
			}
		})
	}
}

// A title no filename can carry is written into the `title` key, whether or not
// the note already carries one.
func TestATitleOnlyTheKeyCanCarryIsSavedThere(t *testing.T) {
	t.Parallel()
	for name, raw := range map[string]string{
		"a note carrying the key":      "---\ntitle: Old\n---\nA measure.\n",
		"a note carrying a heading":    "# Old\n\nA measure.\n",
		"a note named by its filename": "A measure.\n",
	} {
		t.Run(name, func(t *testing.T) {
			v := openChanging(t, map[string]string{"Old.md": raw})

			renamed, err := v.rename().Execute(t.Context(), v.vault, "Old.md", "C#")
			if err != nil {
				t.Fatalf("the rename failed: %v", err)
			}
			if renamed.By != naming.ByFrontmatter {
				t.Errorf("want named by the frontmatter, got %s", renamed.By)
			}
			if got := v.title(t, renamed.Path); got != "C#" {
				t.Errorf("the vault shows the note as %q", got)
			}
		})
	}
}

// A link written with the note's name resolves to the note after it is renamed.
func TestARenamedNoteIsStillReachedByTheLinksThatNameIt(t *testing.T) {
	t.Parallel()
	for name, title := range map[string]string{
		"a title in double brackets":   "Notes [[draft]]",
		"a title carrying a pipe":      "Either|Or",
		"a title carrying a hash":      "Issue #42",
		"a title with nothing awkward": "Thermodynamics",
	} {
		t.Run(name, func(t *testing.T) {
			c := openChanging(t, map[string]string{
				"Entropy.md": "# Entropy\n",
				"Heat.md":    "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n# Heat\n",
			})

			renamed, err := c.rename().Execute(t.Context(), c.vault, "Entropy.md", title)
			if err != nil {
				t.Fatalf("the rename failed: %v", err)
			}
			if !domain.IsNameable(domain.Basename(renamed.Path)) {
				t.Fatalf("no link can be written by the name of %q", renamed.Path)
			}

			found := links(t, c.db, c.vault, "Heat.md")
			if len(found.Links) != 1 {
				t.Fatalf("want the one link, got %+v", found.Links)
			}
			if found.Links[0].To != renamed.Path {
				t.Errorf("the link reaches %q, and the note is at %q", found.Links[0].To, renamed.Path)
			}
		})
	}
}

// A note whose filename is the whole of its naming is moved and not edited, so
// it comes out of a rename with the bytes it went in with.
func TestRenamingByTheFilenameAlonePreservesTheBytesAlone(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Old.md": "A measure.\n"})

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
	t.Parallel()
	c := openChanging(t, map[string]string{"Entropy.md": "---\ntitle: Old\n---\nA measure.\n"})

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
	if body := c.read(t, "Entropy.md"); !strings.Contains(body, "title: Entropy") {
		t.Errorf("the title was not rewritten:\n%s", body)
	}
}

// When the file was already renamed before the note is opened, frontmatter
// that cannot be edited leaves the title unwritten and reports nothing. A
// one-line block and an anchor are both such cases: neither is written, and
// both are rejected by the same guard.
func TestWriteFilenameAsTitle_SkipsUneditableFrontmatter(t *testing.T) {
	t.Parallel()
	for name, raw := range map[string]string{
		"a block written on one line": "---\n{title: Old, id: b}\n---\nA measure.\n",
		"a block carrying an anchor":  "---\ntitle: &t Old\nalias: *t\n---\nA measure.\n",
	} {
		t.Run(name, func(t *testing.T) {
			c := openChanging(t, map[string]string{"Entropy.md": raw})

			if err := c.move().WriteFilenameAsTitle(t.Context(), c.vault, "Entropy.md"); err != nil {
				t.Fatalf("the file had already landed: %v", err)
			}
			if got := c.read(t, "Entropy.md"); got != raw {
				t.Errorf("want the note untouched, got %q", got)
			}
		})
	}
}

func TestRename_RejectsExistingTarget(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"Old.md":     "---\ntitle: Old\n---\nA measure.\n",
		"Entropy.md": "# Entropy\n",
	})

	renamed, err := c.rename().Execute(t.Context(), c.vault, "Old.md", "Entropy")
	if !errors.Is(err, vault.ErrAlreadyExists) {
		t.Fatalf("want ErrAlreadyExists, got %v", err)
	}
	if renamed.Path != "Old.md" {
		t.Errorf("want the note where it still is, got %s", renamed.Path)
	}
	if renamed.Moved != nil {
		t.Errorf("the file did not move, and the answer says %+v", renamed.Moved)
	}
	if body := c.read(t, "Old.md"); body != "---\ntitle: Old\n---\nA measure.\n" {
		t.Errorf("the note was written although the rename failed:\n%s", body)
	}
	if body := c.read(t, "Entropy.md"); body != "# Entropy\n" {
		t.Errorf("the note already there was written over:\n%s", body)
	}
}

// sulking is an index that fails when notified of a file move.
type sulking struct {
	sourceport.SourceRepository
	moveError error
}

func (s sulking) MoveSources(context.Context, domain.VaultID, string, string) error {
	return s.moveError
}

// The result reports the new file location. A move that completed is
// reported regardless of whether the follow-up work fails.
func TestRename_ReturnsMoveWhenFollowUpFails(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Old.md": "# Old\n"})

	sulk := errors.New("the index would not have it")
	deps := note.MoveDeps{
		Readers: vaultreader.Readers{}, Writers: vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()},
		Links: c.db.NoteQueries(), Names: c.db.NoteQueries(),
		Sources: sulking{SourceRepository: c.db.Sources(), moveError: sulk}, Index: c.index, Now: testsupport.NewClock(),
	}
	rename := note.NewRename(deps, note.NewMove(deps), note.OpenGate{})

	renamed, err := rename.Execute(t.Context(), c.vault, "Old.md", "Entropy")
	if !errors.Is(err, sulk) {
		t.Fatalf("want the index's own error, got %v", err)
	}
	if renamed.Path != "Entropy.md" {
		t.Errorf("the file is at Entropy.md and the answer says %q", renamed.Path)
	}
	if renamed.Moved == nil {
		t.Error("the file moved and the answer reports no move")
	}
	if body := c.read(t, "Entropy.md"); body != "# Old\n" {
		t.Errorf("the file is not where the answer says:\n%s", body)
	}
}

func TestRename_RejectsInvalidTitle(t *testing.T) {
	t.Parallel()
	for name, title := range map[string]string{
		"nothing at all":                "",
		"only spaces":                   "   ",
		"only dots":                     "...",
		"only controls":                 "\x00\x01",
		"a line break":                  "one\ntwo",
		"a line break making a heading": "one\n# two",
	} {
		t.Run(name, func(t *testing.T) {
			c := openChanging(t, map[string]string{"Old.md": "# Old\n"})

			renamed, err := c.rename().Execute(t.Context(), c.vault, "Old.md", title)
			if !errors.Is(err, domain.ErrInvalidName) {
				t.Fatalf("want ErrInvalidName, got %v", err)
			}
			if renamed.Path != "" {
				t.Errorf("a rejected rename answered with %q", renamed.Path)
			}
			if body := c.read(t, "Old.md"); body != "# Old\n" {
				t.Errorf("the rejected rename wrote to the note:\n%s", body)
			}
		})
	}
}

// The title is trimmed once, so the name, the note and the answer all say the
// same thing.
func TestRenamingTrimsTheTitleItIsGiven(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Old.md": "# Old\n"})

	renamed, err := c.rename().Execute(t.Context(), c.vault, "Old.md", "  Entropy  ")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Title != "Entropy" {
		t.Errorf("the answer says the note is called %q", renamed.Title)
	}
	if renamed.Path != "Entropy.md" {
		t.Errorf("the note is filed at %q", renamed.Path)
	}
	if got := c.title(t, renamed.Path); got != "Entropy" {
		t.Errorf("the vault shows it as %q", got)
	}
}

// A title longer than a filename will take is cut to make the name, and the
// whole of it goes into the note, where the order of resolution finds it.
func TestRename_LongTitleTruncatesNameAndCachesFullTitle(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Old.md": "A measure.\n"})
	title := strings.TrimSpace(strings.Repeat("disorder ", 20))

	renamed, err := c.rename().Execute(t.Context(), c.vault, "Old.md", title)
	if err != nil {
		t.Fatal(err)
	}
	if len(renamed.Path) >= len(title) {
		t.Errorf("the name was not cut: %s", renamed.Path)
	}
	if renamed.By != naming.ByFrontmatter {
		t.Errorf("want named by the frontmatter, got %s", renamed.By)
	}
	if body := c.read(t, renamed.Path); !strings.Contains(body, "title: "+title) {
		t.Errorf("the whole title is not in the note:\n%s", body)
	}
	if got := c.title(t, renamed.Path); got != title {
		t.Errorf("the vault shows it as %q", got)
	}
}

// The file half of a rename is a move, so a link that stopped resolving is
// repaired by the machinery a move already has.
func TestRenamingRepairsALinkThatStoppedResolving(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
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

// A note the vault does not hold is named as one, and a vault that cannot be
// reached is not.
func TestOnlyAMissingNoteIsNamedAsOne(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Old.md": "# Old\n"})

	if _, err := c.rename().Execute(t.Context(), c.vault, "Missing.md", "Entropy"); !errors.Is(err, edit.ErrNoNote) {
		t.Errorf("want ErrNoNote for a note that is not there, got %v", err)
	}

	elsewhere := c.vault
	elsewhere.Path = filepath.Join(t.TempDir(), "no vault here")
	_, err := c.rename().Execute(t.Context(), elsewhere, "Old.md", "Entropy")
	if err == nil {
		t.Fatal("want the vault's own error")
	}
	if errors.Is(err, edit.ErrNoNote) {
		t.Errorf("a vault that is not there was named as a missing note: %v", err)
	}
}

// Removing a note deletes it from the vault. Removing a note that does not
// exist reports it as missing, not as a vault failure.
func TestRemovingReturnsWhenThereIsNoSuchNote(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Old.md": "# Old\n"})
	remove := note.NewRemove(vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, c.db.NoteQueries(), c.db.Sources(), c.index, note.OpenGate{})

	if _, err := remove.Execute(t.Context(), c.vault, "Missing.md"); !errors.Is(err, edit.ErrNoNote) {
		t.Errorf("want ErrNoNote, got %v", err)
	}
}

// A renamed note is found by the name it was given, from any folder in the
// vault. Whichever of the three carries the name, the note answers to it.
func TestARenamedNoteIsFoundByItsNewName(t *testing.T) {
	t.Parallel()
	for name, raw := range map[string]string{
		"a title in the frontmatter": "---\ntitle: Old\n---\nA measure.\n",
		"a level-one heading":        "# Old\n\nA measure.\n",
		"neither, so the filename":   "A measure.\n",
	} {
		t.Run(name, func(t *testing.T) {
			c := openChanging(t, map[string]string{"physics/Old.md": raw})

			renamed, err := c.rename().Execute(t.Context(), c.vault, "physics/Old.md", "Entropy")
			if err != nil {
				t.Fatal(err)
			}
			if renamed.Path != "physics/Entropy.md" {
				t.Fatalf("the note is filed at %q", renamed.Path)
			}
			if got := c.title(t, renamed.Path); got != "Entropy" {
				t.Errorf("the vault shows the note as %q", got)
			}

			found, err := c.db.NoteQueries().GetNamedPaths(t.Context(), c.vault.ID, "Entropy")
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(found, []string{"physics/Entropy.md"}) {
				t.Errorf("the vault files %v under the name it was given", found)
			}
		})
	}
}

// When the title and the filename are separate, a new title is written into
// the note and the file does not move.
func TestRename_RetitlesInFrontmatterWithoutMovingFile(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		raw   string
		title string
		delta naming.Source
		holds string
	}{
		"a title in the frontmatter": {
			raw:   "---\ntitle: Old\n---\nA measure.\n",
			title: "Entropy",
			delta: naming.ByFrontmatter,
			holds: "title: Entropy",
		},
		"no key, and a title the filename cannot carry": {
			raw:   "# Old\n\nA measure.\n",
			title: "TCP/IP",
			delta: naming.ByFrontmatter,
			holds: "title: TCP/IP",
		},
	} {
		t.Run(name, func(t *testing.T) {
			v := openChanging(t, map[string]string{"Old.md": c.raw})

			renamed, err := v.renameApart().Execute(t.Context(), v.vault, "Old.md", c.title)
			if err != nil {
				t.Fatal(err)
			}
			if renamed.Path != "Old.md" {
				t.Errorf("the note is filed at %q", renamed.Path)
			}
			if renamed.Moved != nil {
				t.Errorf("the file moved: %+v", renamed.Moved)
			}
			if renamed.By != c.delta {
				t.Errorf("want named by %s, got %s", c.delta, renamed.By)
			}
			if body := v.read(t, "Old.md"); !strings.Contains(body, c.holds) {
				t.Errorf("want %q in\n%s", c.holds, body)
			}
			if got := v.title(t, "Old.md"); got != c.title {
				t.Errorf("the vault shows it as %q", got)
			}
			if !isGone(t, v, "Entropy.md") {
				t.Error("the file is at Entropy.md")
			}
		})
	}
}

// A note its filename names carries its name nowhere else, so its file moves
// however a title and a filename are held.
func TestRenamingANoteItsFilenameNamesMovesTheFileEitherWay(t *testing.T) {
	t.Parallel()
	for name, renaming := range map[string]func(changing) note.Rename{
		"one name":   changing.rename,
		"told apart": changing.renameApart,
	} {
		t.Run(name, func(t *testing.T) {
			v := openChanging(t, map[string]string{"Old.md": "A measure.\n"})

			renamed, err := renaming(v).Execute(t.Context(), v.vault, "Old.md", "Entropy")
			if err != nil {
				t.Fatal(err)
			}
			if renamed.Path != "Entropy.md" {
				t.Fatalf("the note is filed at %q", renamed.Path)
			}
			if renamed.By != naming.ByFilename {
				t.Errorf("want named by filename, got %s", renamed.By)
			}
			if body := v.read(t, "Entropy.md"); body != "A measure.\n" {
				t.Errorf("want the note untouched, got %q", body)
			}
			if got := v.title(t, "Entropy.md"); got != "Entropy" {
				t.Errorf("the vault shows it as %q", got)
			}
		})
	}
}

func isGone(t *testing.T, c changing, path string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(c.vault.Path, filepath.FromSlash(path)))
	return errors.Is(err, fs.ErrNotExist)
}

// On a case-sensitive filesystem, a file whose name differs from the target only in case is
// another file, and a rename onto it leaves the note as it was.
func TestRename_RejectsTargetThatDiffersOnlyInCase(t *testing.T) {
	t.Parallel()
	const old = "---\ntitle: rust\n---\nOxidized iron.\n"
	const other = "# Rust\n\nA programming language.\n"
	c := openChanging(t, map[string]string{"rust.md": old, "Rust.md": other})
	// Where the filesystem ignores case, the two names are one file, containing whichever was written last.
	if c.read(t, "rust.md") != old || c.read(t, "Rust.md") != other {
		t.Skip("the filesystem ignores case, so rust.md and Rust.md are one file")
	}

	renamed, err := c.rename().Execute(t.Context(), c.vault, "rust.md", "Rust")

	if !errors.Is(err, vault.ErrAlreadyExists) {
		t.Fatalf("want ErrAlreadyExists, got %v (%+v)", err, renamed)
	}
	if body := c.read(t, "rust.md"); body != old {
		t.Errorf("the note was written although the rename failed:\n%s", body)
	}
	if body := c.read(t, "Rust.md"); body != other {
		t.Errorf("the other note was written over:\n%s", body)
	}
}

// On a filesystem that ignores Unicode normalization, the decomposed and the composed spelling of a
// name are one file, so a rename to the other spelling of the note's own name is not a rename onto
// another file.
func TestRename_AcceptsOtherNormalizationOfOwnFilename(t *testing.T) {
	t.Parallel()
	const decomposed, composed = "Café", "Café"
	c := openChanging(t, map[string]string{decomposed + ".md": "---\ntitle: Bistro\n---\nA place.\n"})
	if _, err := os.Stat(filepath.Join(c.vault.Path, composed+".md")); err != nil {
		t.Skip("the filesystem is case-sensitive")
	}

	renamed, err := c.rename().Execute(t.Context(), c.vault, decomposed+".md", composed)

	if err != nil {
		t.Fatalf("renaming the note to the composed spelling of its own filename failed: %v (%+v)", err, renamed)
	}
	if body := c.read(t, renamed.Path); !strings.Contains(body, "title: "+composed) {
		t.Errorf("the title was not written:\n%s", body)
	}
}
