package load

import (
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"golang.org/x/tools/go/packages"
)

func WriteFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	files["go.mod"] = "module fixture\n\ngo 1.26\n"
	for name, text := range files {
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}
