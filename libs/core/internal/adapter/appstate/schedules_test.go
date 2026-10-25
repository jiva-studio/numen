package appstate_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// What was worked out is kept by the vault it was worked out for, and comes
// back as it was written.
func TestSchedulesAreCachedByVault(t *testing.T) {
	schedules := appstate.SchedulesAt(filepath.Join(t.TempDir(), "flashcards"))
	ctx := t.Context()

	if err := schedules.Write(ctx, "01J8F3K2M9QRSTVWXYZ012", []byte(`{"v":1}`)); err != nil {
		t.Fatal(err)
	}
	got, err := schedules.Read(ctx, "01J8F3K2M9QRSTVWXYZ012")
	if err != nil || string(got) != `{"v":1}` {
		t.Errorf("read back %q, %v", got, err)
	}

	if _, err := schedules.Read(ctx, "01J8F3K2M9QRSTVWXYZ999"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a vault nothing was kept for answers %v", err)
	}
}

// The identity is written into a file's name, so a name that is not one is
func TestAnIdentityThatIsNotOneNamesNoFile(t *testing.T) {
	schedules := appstate.SchedulesAt(filepath.Join(t.TempDir(), "flashcards"))
	for _, id := range []domain.VaultID{"", "../numen.json", "a/b", `a\b`, "one.json"} {
		if err := schedules.Write(t.Context(), id, []byte("{}")); err == nil {
			t.Errorf("wrote under %q", id)
		}
	}
}

// The platform's cache location holds the schedules under numen/flashcards and
// the days of review under numen/flashcards/days.
func TestSchedulesAndCountingStandInThePlatformsCacheLocation(t *testing.T) {
	home := t.TempDir()
	cache := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", cache)
	base, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}

	schedules, err := appstate.OpenSchedules()
	if err != nil {
		t.Fatal(err)
	}
	if err := schedules.Write(t.Context(), "vault-a", []byte("schedules")); err != nil {
		t.Fatal(err)
	}
	counting, err := appstate.OpenCounting()
	if err != nil {
		t.Fatal(err)
	}
	if err := counting.Write(t.Context(), "vault-a", []byte("days")); err != nil {
		t.Fatal(err)
	}

	for path, want := range map[string]string{
		filepath.Join(base, "numen", "flashcards", "vault-a.json"):         "schedules",
		filepath.Join(base, "numen", "flashcards", "days", "vault-a.json"): "days",
	} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Errorf("%s holds %q, %v; want %q", path, got, err, want)
		}
	}
}

// Without a cache location there is nowhere to keep anything, and opening says so.
func TestNoCacheLocationOpensNothing(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	if s, err := appstate.OpenSchedules(); err == nil || s != nil {
		t.Errorf("OpenSchedules gave %v, %v", s, err)
	}
	if s, err := appstate.OpenCounting(); err == nil || s != nil {
		t.Errorf("OpenCounting gave %v, %v", s, err)
	}
}

// A vault whose identity is not one is read as an error.
func TestAnIdentityThatIsNotOneReadsNothing(t *testing.T) {
	schedules := appstate.SchedulesAt(t.TempDir())
	for _, id := range []domain.VaultID{"", "a/b", "one.json"} {
		if _, err := schedules.Read(t.Context(), id); err == nil {
			t.Errorf("read under %q", id)
		}
	}
}

// The folder is made when it is missing, and a folder that cannot be made is an
// error.
func TestAFolderThatCannotBeMadeFailsTheWrite(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	schedules := appstate.SchedulesAt(filepath.Join(file, "flashcards"))
	if err := schedules.Write(t.Context(), "vault-a", []byte("{}")); err == nil {
		t.Error("wrote under a folder that is a file")
	}
}

// A file is written readable by its owner alone.
func TestASavedFileIsTheOwnersAlone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows files have no permission bits to read")
	}
	dir := filepath.Join(t.TempDir(), "flashcards")
	if err := appstate.SchedulesAt(dir).Write(t.Context(), "vault-a", []byte("{}")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "vault-a.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("mode %v", got)
	}
}

// A write that cannot place the file where it belongs fails and leaves nothing
// beside it.
func TestAWriteThatCannotReplaceTheFilePreservesNothingPrior(t *testing.T) {
	dir := t.TempDir()
	occupied := filepath.Join(dir, "vault-a.json")
	if err := os.MkdirAll(filepath.Join(occupied, "inner"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := appstate.SchedulesAt(dir).Write(t.Context(), "vault-a", []byte("{}")); err == nil {
		t.Fatal("replaced a folder with a file")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "vault-a.json" {
		t.Errorf("the folder holds %v", entries)
	}
}

// A folder that cannot be written into fails the write.
func TestAFolderThatCannotBeSavedIntoFailsTheWrite(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a superuser writes into any folder")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	if err := appstate.SchedulesAt(dir).Write(t.Context(), "vault-a", []byte("{}")); err == nil {
		t.Error("wrote into a read-only folder")
	}
}
