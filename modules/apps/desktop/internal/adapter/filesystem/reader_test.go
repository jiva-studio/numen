package filesystem_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/ulid"
)

func walkPaths(t *testing.T, root string) []string {
	t.Helper()
	src, err := filesystem.Open(root, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := src.Walk(t.Context(), func(r domain.FileRef) error {
		got = append(got, r.Path)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	return got
}

func TestWalkReportsEveryNoteAndNothingElse(t *testing.T) {
	got := walkPaths(t, testsupport.VaultDir(t))
	want := []string{
		"Thermodynamics.md",
		"daily/2026-08-15.md",
		"edge/broken-frontmatter.md",
		"edge/broken-links.md",
		"edge/code-fence.md",
		"edge/crlf.md",
		"edge/unicode.md",
		"notes/Entropy.md",
	}
	if !slices.Equal(got, want) {
		t.Errorf("walk found\n  %v\nwant\n  %v", got, want)
	}
}

func TestWalkSkipsTheServiceFolderAndOtherToolsFolders(t *testing.T) {
	for _, p := range walkPaths(t, testsupport.VaultDir(t)) {
		if p[0] == '.' {
			t.Errorf("walk reported %q from a hidden folder", p)
		}
	}
}

func TestWalkPathsAreRelativeAndSlashed(t *testing.T) {
	// The index describes the same note on every platform, so a walk may not
	// leak an absolute path or a backslash into it.
	for _, p := range walkPaths(t, testsupport.VaultDir(t)) {
		if filepath.IsAbs(p) {
			t.Errorf("%q is absolute", p)
		}
		for _, r := range p {
			if r == '\\' {
				t.Errorf("%q contains a backslash", p)
			}
		}
	}
}

func TestWalkReportsSizeAndTime(t *testing.T) {
	src, err := filesystem.Open(testsupport.VaultDir(t), filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := src.Walk(t.Context(), func(r domain.FileRef) error {
		if r.Size <= 0 {
			t.Errorf("%s has size %d", r.Path, r.Size)
		}
		if r.MTime == 0 {
			t.Errorf("%s has no modification time", r.Path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestOpenRejectsWhatIsNotAFolder(t *testing.T) {
	if _, err := filesystem.Open(filepath.Join(testsupport.VaultDir(t), "Thermodynamics.md"), filesystem.Options{}); err == nil {
		t.Error("a file was accepted as a vault")
	}
	if _, err := filesystem.Open(filepath.Join(t.TempDir(), "nope"), filesystem.Options{}); err == nil {
		t.Error("a missing folder was accepted as a vault")
	}
}

func TestReadReturnsTheFile(t *testing.T) {
	src, err := filesystem.Open(testsupport.VaultDir(t), filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := src.Read(t.Context(), "notes/Entropy.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Error("read returned nothing")
	}
}

func TestOpenDoesNotWriteIntoTheFolder(t *testing.T) {
	// Scanning a folder and adding a vault are different acts, and only the
	// second may write.
	dir := t.TempDir()
	if _, err := filesystem.Open(dir, filesystem.Options{}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("opening a folder created %v", entries)
	}
}

func TestInitializeGivesAnIdentityOnceAndKeepsIt(t *testing.T) {
	dir := t.TempDir()

	first, err := filesystem.Initialize(dir, "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !ulid.Valid(first.ID) {
		t.Fatalf("identity %q is not a ULID", first.ID)
	}
	if first.V != 1 {
		t.Errorf("format version = %d", first.V)
	}

	// Re-adding a vault must not mint a new identity: every row in the index
	// points at the old one.
	second, err := filesystem.Initialize(dir, "", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Errorf("identity changed from %s to %s", first.ID, second.ID)
	}
}

func TestInitializeWritesOnlyIntoTheServiceFolder(t *testing.T) {
	dir := t.TempDir()
	if _, err := filesystem.Initialize(dir, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filesystem.DefaultServiceDir {
		t.Errorf("initialize touched %v, want only %s", entries, filesystem.DefaultServiceDir)
	}
}

func TestServiceFolderNameIsASetting(t *testing.T) {
	dir := t.TempDir()
	if _, err := filesystem.Initialize(dir, "_numen", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "_numen", "config.json")); err != nil {
		t.Errorf("configured service folder not used: %v", err)
	}
}

func TestReadConfigDistinguishesMissingFromCorrupt(t *testing.T) {
	dir := t.TempDir()
	if _, err := filesystem.ReadConfig(dir, ""); !errors.Is(err, filesystem.ErrNotAVault) {
		t.Errorf("missing configuration gave %v, want ErrNotAVault", err)
	}

	svc := filepath.Join(dir, filesystem.DefaultServiceDir)
	if err := os.MkdirAll(svc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(svc, "config.json"), []byte(`{"v":1,"id":"nonsense"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// A corrupt identity is not "no vault here": treating it as absent would
	// mint a second identity for a vault that already has one.
	err := func() error { _, e := filesystem.ReadConfig(dir, ""); return e }()
	if err == nil || errors.Is(err, filesystem.ErrNotAVault) {
		t.Errorf("corrupt identity gave %v", err)
	}
	if !errors.Is(err, ulid.ErrInvalid) {
		t.Errorf("error does not say what is wrong: %v", err)
	}
}

func TestWhichExtensionsCountIsASetting(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.md", "b.markdown", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("# x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// The default is markdown alone: a default that guesses widely indexes what
	// the user did not mean.
	if got := walked(t, dir, filesystem.Options{}); !slices.Equal(got, []string{"a.md"}) {
		t.Errorf("default found %v, want [a.md]", got)
	}

	got := walked(t, dir, filesystem.Options{Extensions: []string{".md", ".markdown"}})
	if !slices.Equal(got, []string{"a.md", "b.markdown"}) {
		t.Errorf("configured found %v", got)
	}
}

func TestTheConfiguredServiceFolderIsSkippedEvenWithoutALeadingDot(t *testing.T) {
	// The name is a setting because a leading dot is not free — some sync tools
	// skip hidden directories — so a service folder called _numen must be
	// skipped by its name rather than by its shape.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "_numen"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "_numen", "notes.md"), []byte("# ours\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "real.md"), []byte("# theirs\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := walked(t, dir, filesystem.Options{ServiceDir: "_numen"})
	if !slices.Equal(got, []string{"real.md"}) {
		t.Errorf("walk found %v, want [real.md]", got)
	}
}

func walked(t *testing.T, root string, opts filesystem.Options) []string {
	t.Helper()
	src, err := filesystem.Open(root, opts)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := src.Walk(t.Context(), func(r domain.FileRef) error {
		got = append(got, r.Path)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	return got
}

// A path that leaves the vault is refused rather than resolved. `filepath.Join`
// would clean the dot-dots away and read whatever it landed on, and a caller
// from outside the application is exactly who would try.
func TestAPathThatLeavesTheVaultIsRefused(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.md"), []byte("not yours"), 0o600); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(outside, "vault")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	reader, err := filesystem.Open(root, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"../secret.md",
		"notes/../../secret.md",
		"./notes/../../secret.md",
	} {
		if _, err := reader.Read(t.Context(), path); !errors.Is(err, filesystem.ErrOutside) {
			t.Errorf("read %q: want ErrOutside, got %v", path, err)
		}
		if _, err := reader.Stat(t.Context(), path); err == nil {
			t.Errorf("stat %q: want a refusal", path)
		}
	}
}

// The writer holds the same rules the reader does. Without that, an agent could
// remove the vault's attachments, another tool's state, or the repository the
// vault is kept in — while the reader was already saying those paths do not
// exist.
func TestTheWriterOnlyTouchesNotes(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{".git/config", "photo.png", "notes/keep.md"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte("theirs"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writer, err := filesystem.OpenForWriting(root, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{".git/config", "photo.png"} {
		if err := writer.Write(t.Context(), path, []byte("mine"), domain.FileRef{}); !errors.Is(err, filesystem.ErrNotANote) {
			t.Errorf("write %s: want ErrNotANote, got %v", path, err)
		}
		if err := writer.Remove(t.Context(), path); !errors.Is(err, filesystem.ErrNotANote) {
			t.Errorf("remove %s: want ErrNotANote, got %v", path, err)
		}
		if kept, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(path))); string(kept) != "theirs" {
			t.Errorf("%s was written to anyway", path)
		}
	}

	if err := writer.Write(t.Context(), "notes/keep.md", []byte("mine"), domain.FileRef{}); err != nil {
		t.Errorf("a note is still writable: %v", err)
	}
}

// A folder inside the vault may be a link to somewhere else, and a write
// through it lands outside. The text of the path says nothing about that.
func TestAWriteDoesNotFollowALinkOutOfTheVault(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.md"), []byte("not yours"), 0o600); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("this filesystem does not do symlinks: %v", err)
	}
	writer, err := filesystem.OpenForWriting(root, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if err := writer.Write(t.Context(), "linked/secret.md", []byte("mine"), domain.FileRef{}); !errors.Is(err, filesystem.ErrOutside) {
		t.Errorf("want ErrOutside, got %v", err)
	}
	if kept, _ := os.ReadFile(filepath.Join(outside, "secret.md")); string(kept) != "not yours" {
		t.Error("a file outside the vault was written")
	}
}
