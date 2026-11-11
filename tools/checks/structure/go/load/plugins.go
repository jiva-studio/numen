package load

import (
	"os"
	"testing"

	"golang.org/x/mod/modfile"
)

func ReadGoMod(t *testing.T, path string) *modfile.File {
	t.Helper()
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := modfile.Parse(path, text, nil)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func GetModulePath(file *modfile.File) string {
	if file.Module == nil {
		return ""
	}
	return file.Module.Mod.Path
}
