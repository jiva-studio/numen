package vaultwriter_test

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

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultfixture"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// openWriter opens an empty vault for writing.
func openWriter(t *testing.T) (vaultport.VaultWriter, string) {
	t.Helper()
	root := t.TempDir()
	w, err := vaultwriter.VaultWriters{}.Open(domain.Vault{Path: root})
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
	w, root := openWriter(t)
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
	w, root := openWriter(t)
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
	w, root := openWriter(t)
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

// A vault holds books and whatever else the person filed there, and every one
// of them moves.
func TestAnyFileOfTheVaultMoves(t *testing.T) {
	w, root := openWriter(t)
	ctx := t.Context()

	moves := map[string]string{
		"A Book.epub": "library/A Book.epub",
		"scan.png":    "assets/scan.png",
		"notes.txt":   "assets/notes.txt",
	}
	for from := range moves {
		vaultfixture.PlaceFile(t, root, from)
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
func TestAFolderMovesWithWhatIsBelowIt(t *testing.T) {
	w, root := openWriter(t)
	ctx := t.Context()

	vaultfixture.PlaceFile(t, root, "physics/Entropy.md")
	vaultfixture.PlaceFile(t, root, "physics/deeper/Heat.md")
	vaultfixture.PlaceFile(t, root, "physics/A Book.epub")

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

// A destination that is taken fails, and what was going there stays where
// it is.
func TestAMoveOntoATakenNameIsRefused(t *testing.T) {
	w, root := openWriter(t)
	ctx := t.Context()

	vaultfixture.PlaceFile(t, root, "scan.png")
	vaultfixture.PlaceFile(t, root, "assets/scan.png")

	if err := w.Move(ctx, "scan.png", "assets/scan.png"); !errors.Is(err, vaultport.ErrAlreadyExists) {
		t.Errorf("want ErrAlreadyExists, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "scan.png")); err != nil {
		t.Errorf("the file did not stay where it was: %v", err)
	}
}

// Taking a file off the disk is open to every file the vault holds.
func TestAnyFileOfTheVaultIsRemoved(t *testing.T) {
	w, root := openWriter(t)
	ctx := t.Context()

	for _, path := range []string{"library/A Book.epub", "assets/scan.png", "notes/Entropy.md"} {
		vaultfixture.PlaceFile(t, root, path)
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
	w, root := openWriter(t)
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

	vaultfixture.PlaceFile(t, root, "science/Entropy.md")
	if err := w.MakeFolder(ctx, "science/Entropy.md"); !errors.Is(err, vaultport.ErrAlreadyExists) {
		t.Errorf("a folder over a file: want ErrAlreadyExists, got %v", err)
	}
	if err := w.MakeFolder(ctx, "../outside"); !errors.Is(err, vaultconfig.ErrOutside) {
		t.Errorf("a folder outside the vault: want ErrOutside, got %v", err)
	}
}

// A note kept as a link to another file in the vault is a link after a write,
// and the file at the other end holds the new bytes.
func TestWritingThroughALinkPreservesTheLink(t *testing.T) {
	v := testsupport.NewVault(t, map[string]string{"notes/Real.md": "# Entropy\n"})
	link := filepath.Join(v.Path, "Entropy.md")
	if err := os.Symlink(filepath.Join("notes", "Real.md"), link); err != nil {
		t.Fatal(err)
	}

	writer, err := (vaultwriter.VaultWriters{}).Open(v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(t.Context(), "Entropy.md", []byte("# Entropy\n\nMine.\n"), domain.FileStat{}); err != nil {
		t.Fatal(err)
	}

	pos, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if pos.Mode()&os.ModeSymlink == 0 {
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

// A file is stated out of its metadata. The file here is sixteen megabytes and
// cannot be opened at all, and the answer still comes back.
func TestStatingAFileOpensNothing(t *testing.T) {
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

	reader, err := vaultreader.Open(root, vaultconfig.Options{})
	if err != nil {
		t.Fatal(err)
	}

	stat, err := reader.StatFile(t.Context(), "export.iso")
	if err != nil {
		t.Fatalf("a file that is there was not stated: %v", err)
	}
	if stat.Kind != domain.KindFile || stat.Size != 16<<20 {
		t.Errorf("stated %+v, want a file of %d bytes", stat, 16<<20)
	}

	// Nothing at a path is the other answer, and it is not this one.
	switch _, err := reader.StatFile(t.Context(), "Entropy.md"); {
	case !errors.Is(err, fs.ErrNotExist):
		t.Errorf("want fs.ErrNotExist, got %v", err)
	case errors.Is(err, vaultport.ErrNotANote):
		t.Errorf("a path with nothing at it was called a file: %v", err)
	}
}

// One vault is held by one writer at a time, and holding one says nothing
// about the next.
func TestOneVaultIsHeldByOneWriterAtATime(t *testing.T) {
	writers := vaultwriter.VaultWriters{}
	here := testsupport.NewVault(t, nil)
	elsewhere := testsupport.NewVault(t, nil)

	release, err := writers.Lock(t.Context(), here)
	if err != nil {
		t.Fatal(err)
	}

	waiting, stop := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer stop()
	if _, err := writers.Lock(waiting, here); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a second hold on a held vault: %v", err)
	}
	if other, err := writers.Lock(t.Context(), elsewhere); err != nil {
		t.Fatalf("another vault was held with this one: %v", err)
	} else {
		other()
	}

	release()
	again, err := writers.Lock(t.Context(), here)
	if err != nil {
		t.Fatal(err)
	}
	again()
}

// A note kept as a link may point at another note and nowhere else. The rule is
// asked of where the bytes land, not only of the name they were asked for by.
func TestWritingThroughALinkOutOfBoundsIsRefused(t *testing.T) {
	for _, item := range []struct {
		name  string
		makes string
		// sentinel is what the rejection carries, where the error is the
		// vault's answer about what it holds. A link into the application's
		// own folder fails before that question is reached.
		sentinel error
	}{
		{"the service folder", ".numen/vault.yml", nil},
		{"another tool's folder", ".obsidian/workspace.json", vaultport.ErrNotANote},
		{"a file that is not a note", "attachments/paper.pdf", vaultport.ErrNotANote},
	} {
		t.Run(item.name, func(t *testing.T) {
			root := t.TempDir()
			target := filepath.Join(root, filepath.FromSlash(item.makes))
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, []byte("what was there\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.FromSlash(item.makes), filepath.Join(root, "Note.md")); err != nil {
				t.Fatal(err)
			}

			w, err := vaultwriter.VaultWriters{}.Open(domain.Vault{Path: root})
			if err != nil {
				t.Fatal(err)
			}
			_, err = w.Write(t.Context(), "Note.md", []byte("# Mine\n"), domain.FileStat{})
			switch {
			case err == nil:
				t.Errorf("writing through the link was allowed")
			case item.sentinel != nil && !errors.Is(err, item.sentinel):
				t.Errorf("writing through the link gave %v, want %v", err, item.sentinel)
			}
			raw, readErr := os.ReadFile(target)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(raw) != "what was there\n" {
				t.Errorf("%s was written: %q", item.makes, raw)
			}
		})
	}
}

// A folder does not go inside itself. The window's tree refuses the gesture, and
// a caller that asks for it anyway fails here.
func TestAFolderIsNotMovedInsideItself(t *testing.T) {
	w, root := openWriter(t)
	ctx := t.Context()

	if err := w.Create(ctx, "physics/Entropy.md", []byte("# Entropy\n")); err != nil {
		t.Fatal(err)
	}

	for _, to := range []string{"physics/inner", "physics/inner/deeper", "physics/inner/physics"} {
		if err := w.Move(ctx, "physics", to); !errors.Is(err, vaultport.ErrAlreadyExists) {
			t.Errorf("moving physics to %s gave %v, want ErrAlreadyExists", to, err)
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
	w, root := openWriter(t)
	ctx := t.Context()

	if err := w.Import(ctx, "scans/Cover.png", strings.NewReader("PNG")); err != nil {
		t.Fatalf("the file failed: %v", err)
	}
	target, err := os.ReadFile(filepath.Join(root, "scans", "Cover.png"))
	if err != nil {
		t.Fatalf("the file did not arrive: %v", err)
	}
	if string(target) != "PNG" {
		t.Errorf("the file arrived as %q", target)
	}

	if err := w.Import(ctx, "scans/Cover.png", strings.NewReader("OTHER")); !errors.Is(err, vaultport.ErrAlreadyExists) {
		t.Errorf("a file over a file: want ErrAlreadyExists, got %v", err)
	}
	if err := w.Import(ctx, "../outside.png", strings.NewReader("PNG")); !errors.Is(err, vaultconfig.ErrOutside) {
		t.Errorf("a file outside the vault: want ErrOutside, got %v", err)
	}
	if err := w.Import(ctx, ".git/hooks/pre-commit", strings.NewReader("#!/bin/sh")); err == nil {
		t.Error("a file was brought into a folder another tool acts on")
	}

	// The bytes cross under a name the watcher passes over, and nothing is left
	// beside the file.
	entries, err := os.ReadDir(filepath.Join(root, "scans"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("the folder holds %d files", len(entries))
	}
}

// A folder whose name begins with another folder's is not inside it.
func TestAFolderMovesAdjacentOneWhoseNameItBegins(t *testing.T) {
	w, root := openWriter(t)
	ctx := t.Context()

	if err := w.Create(ctx, "physics/Entropy.md", []byte("# Entropy\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Move(ctx, "physics", "physics-old"); err != nil {
		t.Fatalf("the move failed: %v", err)
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
	w, root := openWriter(t)
	ctx := t.Context()

	target := filepath.Join(root, vaultconfig.DefaultServiceDir, "ocr")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(target, "abc.txt")
	if err := os.WriteFile(filePath, []byte("what was recognised\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The link is written the way a sync tool writes one, relative to the
	if err := os.Symlink(vaultconfig.DefaultServiceDir, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}

	rejected := map[string]error{
		"write":  errorOf(w.Write(ctx, "link/ocr/note.md", []byte("# Mine\n"), domain.FileStat{})),
		"create": w.Create(ctx, "link/ocr/note.md", []byte("# Mine\n")),
		"bring":  w.Import(ctx, "link/ocr/brought.txt", strings.NewReader("brought\n")),
		"folder": w.MakeFolder(ctx, "link/sneak"),
		"remove": w.Remove(ctx, "link/ocr/abc.txt"),
		"move":   w.Move(ctx, "link/ocr/abc.txt", "Stolen.md"),
	}
	for what, err := range rejected {
		if err == nil {
			t.Errorf("%s through the link was allowed", what)
		}
	}

	body, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("the derived file did not survive: %v", err)
	}
	if string(body) != "what was recognised\n" {
		t.Errorf("the derived file holds %q", body)
	}
	if entries, err := os.ReadDir(target); err != nil || len(entries) != 1 {
		t.Errorf("the application's folder holds %v, %v", entries, err)
	}
}

// errorOf is a write's error, where only that is being asked about.
func errorOf(_ domain.FileStat, err error) error { return err }

// A vault may be reached through a link: a home folder on another disk, a
// synced folder, a temporary folder on a machine that keeps them elsewhere.
// Every rule the writer applies is applied to the name on the other side of it.
func TestAVaultReachedThroughALinkIsSavedLikeAnyOther(t *testing.T) {
	physical := filepath.Join(t.TempDir(), "physical")
	if err := os.Mkdir(physical, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "vault")
	if err := os.Symlink(physical, link); err != nil {
		t.Fatal(err)
	}

	w, err := vaultwriter.VaultWriters{}.Open(domain.Vault{Path: link})
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()

	if _, err := w.Write(ctx, "Note.md", []byte("# Note\n"), domain.FileStat{}); err != nil {
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
	w, root := openWriter(t)
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
	target := domain.FileStat{Size: info.Size(), ModTime: info.ModTime()}
	if _, err := w.Write(ctx, name, []byte("# Note\n\nedited\n"), target); err != nil {
		t.Fatalf("the note could not be saved: %v", err)
	}
	if err := w.Import(ctx, stem+".png", strings.NewReader("PNG")); err != nil {
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
	w, root := openWriter(t)
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

// case.
func TestANameAnotherFileExistsAtIsRefused(t *testing.T) {
	w, _ := openWriter(t)
	ctx := t.Context()

	if err := w.Create(ctx, "Note.md", []byte("# Note\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Create(ctx, "Other.md", []byte("# Other\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Move(ctx, "Note.md", "Other.md"); !errors.Is(err, vaultport.ErrAlreadyExists) {
		t.Errorf("want ErrAlreadyExists, got %v", err)
	}
}
