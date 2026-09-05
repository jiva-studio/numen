package filesystem_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// writing opens an empty vault for writing.
func writing(t *testing.T) (port.VaultWriter, string) {
	t.Helper()
	root := t.TempDir()
	w, err := filesystem.VaultWriters{}.Open(domain.Vault{Path: root})
	if err != nil {
		t.Fatal(err)
	}
	return w, root
}

// A folder something else reads is a folder that acts on what it finds. An agent
// working the vault names where a note goes, and a note that lands in one of
// those is a file the index does not know about, that nothing on screen shows,
// and that another program obeys.
func TestNothingIsMovedIntoAnotherToolsFolder(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	if err := w.Create(ctx, "SKILL.md", []byte("# Skill\n")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scan.png"), []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, to := range []string{
		".claude/skills/evil/SKILL.md",
		".claude/agents/evil.md",
		".git/hooks/SKILL.md",
		".obsidian/SKILL.md",
		".trash/../.claude/planted.md",
		"notes/.claude/skills/evil/SKILL.md",
		"notes/.git/hooks/SKILL.md",
	} {
		for _, from := range []string{"SKILL.md", "scan.png"} {
			if err := w.Move(ctx, from, to); err == nil {
				t.Errorf("moving %s to %s was allowed", from, to)
			}
		}
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(to))); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s was written: %v", to, err)
		}
	}

	// Making a folder answers to the same rule.
	if err := w.MakeFolder(ctx, ".git/hooks"); err == nil {
		t.Error("a folder was made where another tool keeps its state")
	}
}

// Where a note goes when it is taken out of the vault's sight is deliberately not
// a note-place, and moving one there is what removal does.
func TestANoteCanStillBeMovedOutOfSight(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	if err := w.Create(ctx, "Entropy.md", []byte("# Entropy\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Move(ctx, "Entropy.md", ".trash/Entropy.md"); err != nil {
		t.Fatalf("a note could not be taken out of sight: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".trash", "Entropy.md")); err != nil {
		t.Errorf("the note is not in the trash: %v", err)
	}
}

// A note moved to a place a note may live is an ordinary move.
func TestANoteMovesWhereANoteMayLive(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	if err := w.Create(ctx, "Entropy.md", []byte("# Entropy\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Move(ctx, "Entropy.md", "physics/Entropy.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "physics", "Entropy.md")); err != nil {
		t.Errorf("the note did not move: %v", err)
	}
}

// laid puts a file in the vault behind the writer's back, for the kinds the
// writer does not create.
func laid(t *testing.T, root, path string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("theirs"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A vault holds books and whatever else the person filed there, and every one
// of them moves.
func TestAnyFileOfTheVaultMoves(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	moves := map[string]string{
		"A Book.epub": "library/A Book.epub",
		"scan.png":    "assets/scan.png",
		"notes.txt":   "assets/notes.txt",
	}
	for from := range moves {
		laid(t, root, from)
	}

	for from, to := range moves {
		if err := w.Move(ctx, from, to); err != nil {
			t.Errorf("move %s: %v", from, err)
			continue
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(to))); err != nil {
			t.Errorf("%s is not at %s: %v", from, to, err)
		}
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(from))); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s is still where it was: %v", from, err)
		}
	}
}

// A folder moves whole, and everything under it arrives with it.
func TestAFolderMovesWithWhatIsUnderIt(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	laid(t, root, "physics/Entropy.md")
	laid(t, root, "physics/deeper/Heat.md")
	laid(t, root, "physics/A Book.epub")

	if err := w.Move(ctx, "physics", "science/physics"); err != nil {
		t.Fatalf("the folder did not move: %v", err)
	}
	for _, path := range []string{"science/physics/Entropy.md", "science/physics/deeper/Heat.md", "science/physics/A Book.epub"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			t.Errorf("%s did not arrive: %v", path, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, "physics")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the folder is still where it was: %v", err)
	}
}

// A destination that is taken is refused, and what was going there stays where
// it is.
func TestAMoveOntoATakenNameIsRefused(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	laid(t, root, "scan.png")
	laid(t, root, "assets/scan.png")

	if err := w.Move(ctx, "scan.png", "assets/scan.png"); !errors.Is(err, port.ErrOccupied) {
		t.Errorf("want ErrOccupied, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "scan.png")); err != nil {
		t.Errorf("the file did not stay where it was: %v", err)
	}
}

// Taking a file off the disk is open to every file the vault holds.
func TestAnyFileOfTheVaultIsRemoved(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	for _, path := range []string{"library/A Book.epub", "assets/scan.png", "notes/Entropy.md"} {
		laid(t, root, path)
		if err := w.Remove(ctx, path); err != nil {
			t.Errorf("remove %s: %v", path, err)
		}
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s is still there: %v", path, err)
		}
	}
}

// A folder is made on its own, with the folders above it, and one that is
// already there is the outcome that was asked for.
func TestAFolderIsMade(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	if err := w.MakeFolder(ctx, "science/physics"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(root, "science", "physics"))
	if err != nil {
		t.Fatalf("the folder was not made: %v", err)
	}
	if !info.IsDir() {
		t.Error("what was made is not a folder")
	}
	if err := w.MakeFolder(ctx, "science/physics"); err != nil {
		t.Errorf("making it again: %v", err)
	}

	laid(t, root, "science/Entropy.md")
	if err := w.MakeFolder(ctx, "science/Entropy.md"); !errors.Is(err, port.ErrOccupied) {
		t.Errorf("a folder over a file: want ErrOccupied, got %v", err)
	}
	if err := w.MakeFolder(ctx, "../outside"); !errors.Is(err, filesystem.ErrOutside) {
		t.Errorf("a folder outside the vault: want ErrOutside, got %v", err)
	}
}

// A note kept as a link to another file in the vault is a link after a write,
// and the file at the other end holds the new bytes.
func TestWritingThroughALinkLeavesTheLink(t *testing.T) {
	v := testsupport.NewVault(t, map[string]string{"notes/Real.md": "# Entropy\n"})
	link := filepath.Join(v.Path, "Entropy.md")
	if err := os.Symlink(filepath.Join("notes", "Real.md"), link); err != nil {
		t.Fatal(err)
	}

	writer, err := (filesystem.VaultWriters{}).Open(v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(t.Context(), "Entropy.md", []byte("# Entropy\n\nMine.\n"), domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	at, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if at.Mode()&os.ModeSymlink == 0 {
		t.Error("the link was replaced by a file of its own")
	}
	body, err := os.ReadFile(filepath.Join(v.Path, "notes", "Real.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "# Entropy\n\nMine.\n" {
		t.Errorf("the note the link points at was not written:\n%s", body)
	}
}

// What a vault leaves alone is answered for out of the file's metadata. The
// file here is sixteen megabytes and cannot be opened at all, and the answer
// still comes back.
func TestAskingAboutAFileTheVaultLeavesAloneOpensNothing(t *testing.T) {
	root := t.TempDir()
	export, err := os.OpenFile(filepath.Join(root, "export.iso"), os.O_CREATE|os.O_WRONLY, 0o000)
	if err != nil {
		t.Fatal(err)
	}
	if err := export.Truncate(16 << 20); err != nil {
		t.Fatal(err)
	}
	if err := export.Close(); err != nil {
		t.Fatal(err)
	}

	reader, err := filesystem.Open(root, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}

	switch _, err := reader.Stat(t.Context(), "export.iso"); {
	case !errors.Is(err, port.ErrNotANote):
		t.Errorf("want ErrNotANote, got %v", err)
	case errors.Is(err, fs.ErrNotExist):
		t.Errorf("a file that is there was called missing: %v", err)
	}

	// Nothing at a path is the other answer, and it is not this one.
	switch _, err := reader.Stat(t.Context(), "Entropy.md"); {
	case !errors.Is(err, fs.ErrNotExist):
		t.Errorf("want fs.ErrNotExist, got %v", err)
	case errors.Is(err, port.ErrNotANote):
		t.Errorf("a path with nothing at it was called a file: %v", err)
	}
}

// One vault is held by one writer at a time, and holding one says nothing
// about the next.
func TestOneVaultIsHeldByOneWriterAtATime(t *testing.T) {
	writers := filesystem.VaultWriters{}
	here := testsupport.NewVault(t, nil)
	elsewhere := testsupport.NewVault(t, nil)

	release, err := writers.Hold(t.Context(), here)
	if err != nil {
		t.Fatal(err)
	}

	waiting, stop := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer stop()
	if _, err := writers.Hold(waiting, here); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a second hold on a held vault: %v", err)
	}
	if other, err := writers.Hold(t.Context(), elsewhere); err != nil {
		t.Fatalf("another vault was held with this one: %v", err)
	} else {
		other()
	}

	release()
	again, err := writers.Hold(t.Context(), here)
	if err != nil {
		t.Fatal(err)
	}
	again()
}

// A note kept as a link may point at another note and nowhere else. The rule is
// asked of where the bytes land, not only of the name they were asked for by.
func TestWritingThroughALinkOutOfBoundsIsRefused(t *testing.T) {
	for _, at := range []struct {
		name  string
		makes string
		// sentinel is what the refusal carries, where the refusal is the
		// vault's answer about what it holds. A link into the application's
		// own folder is refused before that question is reached.
		sentinel error
	}{
		{"the service folder", ".numen/vault.yml", nil},
		{"another tool's folder", ".obsidian/workspace.json", port.ErrNotANote},
		{"a file that is not a note", "attachments/paper.pdf", port.ErrNotANote},
	} {
		t.Run(at.name, func(t *testing.T) {
			root := t.TempDir()
			held := filepath.Join(root, filepath.FromSlash(at.makes))
			if err := os.MkdirAll(filepath.Dir(held), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(held, []byte("what was there\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.FromSlash(at.makes), filepath.Join(root, "Note.md")); err != nil {
				t.Fatal(err)
			}

			w, err := filesystem.VaultWriters{}.Open(domain.Vault{Path: root})
			if err != nil {
				t.Fatal(err)
			}
			_, err = w.Write(t.Context(), "Note.md", []byte("# Mine\n"), domain.Fingerprint{})
			switch {
			case err == nil:
				t.Errorf("writing through the link was allowed")
			case at.sentinel != nil && !errors.Is(err, at.sentinel):
				t.Errorf("writing through the link gave %v, want %v", err, at.sentinel)
			}
			raw, readErr := os.ReadFile(held)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(raw) != "what was there\n" {
				t.Errorf("%s was written: %q", at.makes, raw)
			}
		})
	}
}

// A folder does not go inside itself. The window's tree refuses the gesture, and
// a caller that asks for it anyway is refused here.
func TestAFolderIsNotMovedInsideItself(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	if err := w.Create(ctx, "physics/Entropy.md", []byte("# Entropy\n")); err != nil {
		t.Fatal(err)
	}

	for _, to := range []string{"physics/inner", "physics/inner/deeper", "physics/inner/physics"} {
		if err := w.Move(ctx, "physics", to); !errors.Is(err, port.ErrOccupied) {
			t.Errorf("moving physics to %s gave %v, want ErrOccupied", to, err)
		}
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(to))); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s was made: %v", to, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "physics", "Entropy.md")); err != nil {
		t.Errorf("the folder did not stay where it was: %v", err)
	}
}

// A file handed to the window is any kind of file, and it lands under the name
// it was handed over with.
func TestAFileIsBroughtInFromOutside(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	if err := w.Bring(ctx, "scans/Cover.png", strings.NewReader("PNG")); err != nil {
		t.Fatalf("the file was refused: %v", err)
	}
	held, err := os.ReadFile(filepath.Join(root, "scans", "Cover.png"))
	if err != nil {
		t.Fatalf("the file did not arrive: %v", err)
	}
	if string(held) != "PNG" {
		t.Errorf("the file arrived as %q", held)
	}

	if err := w.Bring(ctx, "scans/Cover.png", strings.NewReader("OTHER")); !errors.Is(err, port.ErrOccupied) {
		t.Errorf("a file over a file: want ErrOccupied, got %v", err)
	}
	if err := w.Bring(ctx, "../outside.png", strings.NewReader("PNG")); !errors.Is(err, filesystem.ErrOutside) {
		t.Errorf("a file outside the vault: want ErrOutside, got %v", err)
	}
	if err := w.Bring(ctx, ".git/hooks/pre-commit", strings.NewReader("#!/bin/sh")); err == nil {
		t.Error("a file was brought into a folder another tool acts on")
	}

	// The bytes cross under a name the watcher passes over, and nothing is left
	// beside the file.
	beside, err := os.ReadDir(filepath.Join(root, "scans"))
	if err != nil {
		t.Fatal(err)
	}
	if len(beside) != 1 {
		t.Errorf("the folder holds %d files", len(beside))
	}
}

// A folder whose name begins with another folder's is not inside it.
func TestAFolderMovesBesideOneWhoseNameItBegins(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	if err := w.Create(ctx, "physics/Entropy.md", []byte("# Entropy\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Move(ctx, "physics", "physics-old"); err != nil {
		t.Fatalf("the move was refused: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "physics-old", "Entropy.md")); err != nil {
		t.Errorf("the folder did not arrive: %v", err)
	}
}

// A link inside the vault is a second name for the folder it points at, and a
// link pointing at the application's own folder is a second name for that. Not
// one of the writer's doors opens through it: a sync tool or an unpacked
// archive can leave such a link, and a note written through it would land on
// the application's own state.
func TestWritingThroughALinkIntoTheServiceFolderIsRefused(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	held := filepath.Join(root, filesystem.DefaultServiceDir, "ocr")
	if err := os.MkdirAll(held, 0o755); err != nil {
		t.Fatal(err)
	}
	kept := filepath.Join(held, "abc.txt")
	if err := os.WriteFile(kept, []byte("what was recognised\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The link is written the way a sync tool writes one, relative to the
	// folder it stands in.
	if err := os.Symlink(filesystem.DefaultServiceDir, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}

	refused := map[string]error{
		"write":  errorOf(w.Write(ctx, "link/ocr/note.md", []byte("# Mine\n"), domain.Fingerprint{})),
		"create": w.Create(ctx, "link/ocr/note.md", []byte("# Mine\n")),
		"bring":  w.Bring(ctx, "link/ocr/brought.txt", strings.NewReader("brought\n")),
		"folder": w.MakeFolder(ctx, "link/sneak"),
		"remove": w.Remove(ctx, "link/ocr/abc.txt"),
		"move":   w.Move(ctx, "link/ocr/abc.txt", "Stolen.md"),
	}
	for what, err := range refused {
		if err == nil {
			t.Errorf("%s through the link was allowed", what)
		}
	}

	body, err := os.ReadFile(kept)
	if err != nil {
		t.Fatalf("the derived file did not survive: %v", err)
	}
	if string(body) != "what was recognised\n" {
		t.Errorf("the derived file holds %q", body)
	}
	if entries, err := os.ReadDir(held); err != nil || len(entries) != 1 {
		t.Errorf("the application's folder holds %v, %v", entries, err)
	}
}

// errorOf is a write's error, where only that is being asked about.
func errorOf(_ domain.Fingerprint, err error) error { return err }

// A vault may be reached through a link: a home folder on another disk, a
// synced folder, a temporary folder on a machine that keeps them elsewhere.
// Every rule the writer applies is applied to the name on the other side of it.
func TestAVaultReachedThroughALinkIsWrittenLikeAnyOther(t *testing.T) {
	physical := filepath.Join(t.TempDir(), "physical")
	if err := os.Mkdir(physical, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "vault")
	if err := os.Symlink(physical, link); err != nil {
		t.Fatal(err)
	}

	w, err := filesystem.VaultWriters{}.Open(domain.Vault{Path: link})
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()

	if _, err := w.Write(ctx, "Note.md", []byte("# Note\n"), domain.Fingerprint{}); err != nil {
		t.Fatalf("the note could not be written: %v", err)
	}
	if err := w.Move(ctx, "Note.md", "Renamed.md"); err != nil {
		t.Fatalf("the note could not be renamed: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(physical, "Renamed.md"))
	if err != nil {
		t.Fatalf("the note did not land in the vault: %v", err)
	}
	if string(body) != "# Note\n" {
		t.Errorf("the note holds %q", body)
	}
}

// A person names a note in their own editor, and a name the filesystem accepts
// is a name this vault saves. The temporary file written beside it carries a
// leading dot and a suffix of its own, and the whole of that has to fit as well.
func TestANoteWithALongNameIsSaved(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	// 249 bytes, which is longer than a filesystem takes once twelve more are
	// put in front of and behind it.
	stem := strings.Repeat("з", 123)
	name := stem + ".md"

	if err := w.Create(ctx, name, []byte("# Note\n")); err != nil {
		t.Fatalf("the note could not be created: %v", err)
	}
	info, err := os.Stat(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	held := domain.Fingerprint{Size: info.Size(), ModTime: info.ModTime()}
	if _, err := w.Write(ctx, name, []byte("# Note\n\nedited\n"), held); err != nil {
		t.Fatalf("the note could not be saved: %v", err)
	}
	if err := w.Bring(ctx, stem+".png", strings.NewReader("PNG")); err != nil {
		t.Fatalf("the file could not be brought in: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "# Note\n\nedited\n" {
		t.Errorf("the note holds %q", body)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("the vault holds %d files, want the note and the attachment", len(entries))
	}
}

// Correcting a note's capitalisation, or the spelling of an accent in it, is a
// move onto a name a filesystem that tells neither apart already answers with
// the note itself. What the writer sees is two names for one file, which is
// what a filesystem that does tell them apart is given a link to make.
func TestANameThatIsAlreadyThisFileIsNotTaken(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	if err := w.Create(ctx, "note.md", []byte("# Note\n")); err != nil {
		t.Fatal(err)
	}
	if !sameFile(t, filepath.Join(root, "note.md"), filepath.Join(root, "Note.md")) {
		if err := os.Link(filepath.Join(root, "note.md"), filepath.Join(root, "Note.md")); err != nil {
			t.Fatal(err)
		}
	}

	if err := w.Move(ctx, "note.md", "Note.md"); err != nil {
		t.Fatalf("the note could not be renamed onto its own file: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(root, "Note.md"))
	if err != nil {
		t.Fatalf("the note is not at the name it was given: %v", err)
	}
	if string(body) != "# Note\n" {
		t.Errorf("the note holds %q", body)
	}
	// A filesystem that tells the two names apart is asked which of them it
	// wrote down: opening the note by either name says nothing about that.
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(entries, func(e os.DirEntry) bool { return e.Name() == "Note.md" }) {
		t.Errorf("the vault holds %v, and the note is filed under none of it", entries)
	}
}

// sameFile says whether two names are already one file. A filesystem that tells
// capitalisation apart answers no for two spellings of a name it has one of.
func sameFile(t *testing.T, one, other string) bool {
	t.Helper()
	first, err := os.Lstat(one)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.Lstat(other)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	return os.SameFile(first, second)
}

// A name another file stands at is taken, whatever the filesystem does with
// case.
func TestANameAnotherFileStandsAtIsRefused(t *testing.T) {
	w, _ := writing(t)
	ctx := t.Context()

	if err := w.Create(ctx, "Note.md", []byte("# Note\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Create(ctx, "Other.md", []byte("# Other\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Move(ctx, "Note.md", "Other.md"); !errors.Is(err, port.ErrOccupied) {
		t.Errorf("want ErrOccupied, got %v", err)
	}
}
