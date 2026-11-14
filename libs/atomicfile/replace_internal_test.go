package atomicfile

import (
	"errors"
	"io/fs"
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func openTestRoot(t *testing.T, dir string) *os.Root {
	t.Helper()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			t.Error(err)
		}
	})
	return root
}

func TestGetTempPatternFitsWithinNameMaxOnRuneBoundaries(t *testing.T) {
	exact := strings.Repeat("a", 243)
	if got := getTempPattern(exact); got != "."+exact+".*" {
		t.Fatalf("getTempPattern(243 bytes) length = %d, want %d", len(got), len(exact)+3)
	}

	overByOne := strings.Repeat("a", 244)
	if got := getTempPattern(overByOne); got != "."+exact+".*" {
		t.Fatalf("getTempPattern(244 bytes) = %q, want 243-byte base", got)
	}

	multiByte := strings.Repeat("a", 242) + "é"
	if got := getTempPattern(multiByte); got != "."+strings.Repeat("a", 242)+".*" {
		t.Fatalf("getTempPattern(multi-byte boundary) = %q", got)
	}
}

func TestCreateTempFileSkipsExistingNamesAndHonorsAttemptLimit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/.notes.txt.1", []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	root := openTestRoot(t, dir)

	step := 0
	file, name, err := createTempFile(root, ".", ".notes.txt.*", func() string {
		step++
		if step == 1 {
			return "1"
		}
		return "2"
	})
	if err != nil {
		t.Fatalf("createTempFile after collision: %v", err)
	}
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if name != ".notes.txt.2" {
		t.Fatalf("createTempFile name = %q, want .notes.txt.2", name)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("createTempFile mode = %v, want 0o600", info.Mode().Perm())
	}
	if getRandomSuffix() == "" {
		t.Fatal("getRandomSuffix returned an empty string")
	}

	attempts := 0
	_, _, err = createTempFile(root, ".", ".notes.txt.*", func() string {
		attempts++
		return "1"
	})
	if err == nil || attempts != 10_000 {
		t.Fatalf("createTempFile exhausted err = %v after %d attempts, want error after 10000", err, attempts)
	}
}

func TestReplacePropagatesSyncStatCloseRenameAndRemoveFailures(t *testing.T) {
	errSync := errors.New("sync failed")
	errRename := errors.New("rename failed")

	t.Run("temp file sync error", func(t *testing.T) {
		root := openTestRoot(t, t.TempDir())
		calls := 0
		_, err := replace(root, "notes.txt", strings.NewReader("data"), 0o600, root.Rename, func(*os.File) error {
			calls++
			if calls == 1 {
				return errSync
			}
			return nil
		})
		if !errors.Is(err, errSync) {
			t.Fatalf("replace sync error = %v, want %v", err, errSync)
		}
	})

	t.Run("temp file stat chmod and close error", func(t *testing.T) {
		root := openTestRoot(t, t.TempDir())
		_, err := replace(root, "notes.txt", strings.NewReader("data"), 0o600, root.Rename, func(f *os.File) error {
			return f.Close()
		})
		if !errors.Is(err, fs.ErrClosed) {
			t.Fatalf("replace closed temp file error = %v, want %v", err, fs.ErrClosed)
		}
	})

	t.Run("rename and cleanup remove error", func(t *testing.T) {
		root := openTestRoot(t, t.TempDir())
		_, err := replace(root, "notes.txt", strings.NewReader("data"), 0o600, func(string, string) error {
			if closeErr := root.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
			return errRename
		}, (*os.File).Sync)
		if !errors.Is(err, errRename) || !errors.Is(err, os.ErrClosed) {
			t.Fatalf("replace rename+remove error = %v, want both %v and %v", err, errRename, os.ErrClosed)
		}
	})

	t.Run("directory sync error", func(t *testing.T) {
		root := openTestRoot(t, t.TempDir())
		calls := 0
		_, err := replace(root, "notes.txt", strings.NewReader("data"), 0o600, root.Rename, func(*os.File) error {
			calls++
			if calls == 2 {
				return errSync
			}
			return nil
		})
		if !errors.Is(err, errSync) {
			t.Fatalf("replace directory sync error = %v, want %v", err, errSync)
		}
	})
}

func TestSyncDirectoryAndIgnoreSyncError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(dir+"/inner", 0o755); err != nil {
		t.Fatal(err)
	}
	root := openTestRoot(t, dir)
	errDisk := errors.New("disk I/O error")

	if err := syncDirectory(root, "inner", "windows", func(*os.File) error { return errDisk }); err != nil {
		t.Fatalf("syncDirectory on windows = %v, want nil", err)
	}
	if err := syncDirectory(root, "absent", "linux", func(*os.File) error { return errDisk }); err != nil {
		t.Fatalf("syncDirectory on missing folder = %v, want nil", err)
	}
	if err := syncDirectory(root, "inner", "linux", func(*os.File) error { return syscall.EINVAL }); err != nil {
		t.Fatalf("syncDirectory EINVAL = %v, want nil", err)
	}
	if err := syncDirectory(root, "inner", "linux", func(*os.File) error { return errors.ErrUnsupported }); err != nil {
		t.Fatalf("syncDirectory ErrUnsupported = %v, want nil", err)
	}
	if err := syncDirectory(root, "inner", "linux", func(*os.File) error { return errDisk }); !errors.Is(err, errDisk) {
		t.Fatalf("syncDirectory disk error = %v, want %v", err, errDisk)
	}

	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := SyncDirectory(root, "inner"); !errors.Is(err, os.ErrClosed) {
			t.Fatalf("SyncDirectory on closed root = %v, want %v", err, os.ErrClosed)
		}
	}
}
