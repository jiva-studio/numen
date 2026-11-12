package settings_test

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
)

// A file without a size uses the desktop's scale; unsetting GDK_DPI_SCALE keeps
// the tests independent of the session environment.
func TestMain(m *testing.M) {
	if err := os.Unsetenv("GDK_DPI_SCALE"); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func openAt(path string) (settings.File, error) {
	target := settings.DefaultFile()
	err := settings.NewFiles().OpenAt(path, &target)
	return target, err
}

func newDocument() *settings.File {
	target := settings.DefaultFile()
	return &target
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
