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
	src, err := filesystem.Open(root, filesystem.DefaultServiceDir)
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
	src, err := filesystem.Open(testsupport.VaultDir(t), filesystem.DefaultServiceDir)
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
	if _, err := filesystem.Open(filepath.Join(testsupport.VaultDir(t), "Thermodynamics.md"), ""); err == nil {
		t.Error("a file was accepted as a vault")
	}
	if _, err := filesystem.Open(filepath.Join(t.TempDir(), "nope"), ""); err == nil {
		t.Error("a missing folder was accepted as a vault")
	}
}

func TestReadReturnsTheFile(t *testing.T) {
	src, err := filesystem.Open(testsupport.VaultDir(t), "")
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
	if _, err := filesystem.Open(dir, ""); err != nil {
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
