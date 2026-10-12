//go:build unix

package container

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestTheFolderOfTheIndexIsOpenToEveryoneTheUmaskAllows(t *testing.T) {
	isolate(t)
	umask := syscall.Umask(0)
	syscall.Umask(umask)

	for name, cfg := range map[string]Config{
		"named":   {IndexPath: filepath.Join(t.TempDir(), "named", "index.db")},
		"default": {},
	} {
		got, err := cfg.IndexPathOrDefault()
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(filepath.Dir(got))
		if err != nil {
			t.Fatal(err)
		}
		if want := os.FileMode(0o755) &^ os.FileMode(umask); info.Mode().Perm() != want {
			t.Errorf("%s: the folder of the index is %v, want %v", name, info.Mode().Perm(), want)
		}
	}
}
