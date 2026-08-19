package filesystem_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
)

func store(t *testing.T) (*filesystem.Derived, string) {
	t.Helper()
	root := t.TempDir()
	derived, err := filesystem.OpenDerived(root, filesystem.Options{}, filesystem.OCRDir)
	if err != nil {
		t.Fatal(err)
	}
	return derived, root
}

func TestOpeningTheStoreWritesNothing(t *testing.T) {
	// The application looks at every vault it holds. One that puts a folder in
	// each of them has changed a vault by being asked about it.
	derived, root := store(t)
	_ = derived

	if _, err := os.Stat(filepath.Join(root, filesystem.DefaultServiceDir)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("opening the store made %s: %v", filesystem.DefaultServiceDir, err)
	}
}

func TestWhatIsWrittenIsRead(t *testing.T) {
	derived, root := store(t)
	ctx := t.Context()

	if err := derived.Write(ctx, "ocr/abc.txt", []byte("first")); err != nil {
		t.Fatal(err)
	}
	if got, err := derived.Read(ctx, "ocr/abc.txt"); err != nil || string(got) != "first" {
		t.Fatalf("read %q, %v", got, err)
	}

	// Writing again replaces: one recognition run twice is the same bytes.
	if err := derived.Write(ctx, "ocr/abc.txt", []byte("second")); err != nil {
		t.Fatal(err)
	}
	if got, _ := derived.Read(ctx, "ocr/abc.txt"); string(got) != "second" {
		t.Errorf("read %q after replacing", got)
	}

	if err := derived.Append(ctx, "ocr/abc.txt", []byte(" and more")); err != nil {
		t.Fatal(err)
	}
	if got, _ := derived.Read(ctx, "ocr/abc.txt"); string(got) != "second and more" {
		t.Errorf("read %q after appending", got)
	}

	at := filepath.Join(root, filesystem.DefaultServiceDir, filesystem.OCRDir, "abc.txt")
	if _, err := os.Stat(at); err != nil {
		t.Errorf("the file is not where the store says it is: %v", err)
	}
}

func TestAppendingToNothingBeginsIt(t *testing.T) {
	derived, _ := store(t)

	if err := derived.Append(t.Context(), "ocr/part.txt", []byte("one")); err != nil {
		t.Fatal(err)
	}
	if got, _ := derived.Read(t.Context(), "ocr/part.txt"); string(got) != "one" {
		t.Errorf("read %q", got)
	}
}

func TestANameWithNothingUnderItIsAnAnswer(t *testing.T) {
	// The folder is on the person's disk and they may empty it. That is a thing
	// that happened, not a failure of the application.
	derived, _ := store(t)

	if _, err := derived.Read(t.Context(), "ocr/gone.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("reading nothing gave %v, want fs.ErrNotExist", err)
	}
	if err := derived.Remove(t.Context(), "ocr/gone.txt"); err != nil {
		t.Errorf("removing nothing gave %v, want it to be the outcome asked for", err)
	}
}

func TestTheStoreCannotNameTheVaultsIdentity(t *testing.T) {
	// Every name a store takes begins with its own area, and the vault's
	// identity is in the folder and in no area. So the file every index row
	// points at is not a name this can express: not by naming it, and not by
	// climbing out of the area to reach it.
	derived, root := store(t)
	identity := filepath.Join(root, filesystem.DefaultServiceDir, "config.json")
	if err := os.MkdirAll(filepath.Dir(identity), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(identity, []byte(`{"v":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		"ocr/../config.json",
		"ocr/../../.numen/config.json",
		"config.json",
	} {
		if err := derived.Write(t.Context(), name, []byte("overwritten")); err == nil {
			t.Errorf("the store wrote through %q", name)
		}
	}

	if got, err := os.ReadFile(identity); err != nil || string(got) != `{"v":1}` {
		t.Errorf("the vault's identity is now %q, %v", got, err)
	}
}

func TestTheStoreDoesNotWriteThroughALinkOutOfItself(t *testing.T) {
	// A name in the store may be a link that a person, a sync client or another
	// tool put there. Following it would read a note back as a recognition, or
	// write a recognition over a note.
	derived, root := store(t)
	note := filepath.Join(root, "Note.md")
	if err := os.WriteFile(note, []byte("# Note\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	area := filepath.Join(root, filesystem.DefaultServiceDir, filesystem.OCRDir)
	if err := os.MkdirAll(area, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(note, filepath.Join(area, "linked.txt")); err != nil {
		t.Skipf("this filesystem has no links: %v", err)
	}

	if err := derived.Write(t.Context(), "ocr/linked.txt", []byte("overwritten")); err == nil {
		t.Error("the store wrote through a link out of itself")
	}
	if _, err := derived.Read(t.Context(), "ocr/linked.txt"); err == nil {
		t.Error("the store read through a link out of itself")
	}
	if got, _ := os.ReadFile(note); string(got) != "# Note\n" {
		t.Errorf("the note is now %q", got)
	}
}

func TestAStoreIsOneFolder(t *testing.T) {
	root := t.TempDir()
	for _, area := range []string{"a/b", "..", ".", `a\b`} {
		if _, err := filesystem.OpenDerived(root, filesystem.Options{}, area); err == nil {
			t.Errorf("opened a store on %q", area)
		}
	}
}
