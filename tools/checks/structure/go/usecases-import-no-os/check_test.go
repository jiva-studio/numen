// Package usecasesimportnoos is the check `go/usecases-import-no-os`, run on the tree with the planted violations that prove it.
package usecasesimportnoos

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// TestUseCasesImportNoOSFilepathOrFS verifies that usecase/ never imports path/filepath or io/fs.
func TestUseCasesImportNoOSFilepathOrFS(t *testing.T) {
	wrong, read, err := findOSPathImports("../usecase")
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}
	if read == 0 {
		t.Fatal("no use case file read: the rule is checked against nothing")
	}
}

// The rule rejects path/filepath and io/fs, a renamed import included, and passes path, os and a test file.
func TestOSPathRule_RejectsFilepathAndFS(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"note/note.go":      "package note\n\nimport (\n\t\"path\"\n\tstdfs \"io/fs\"\n\t\"path/filepath\"\n)\n",
		"note/other.go":     "package note\n\nimport \"os\"\n",
		"note/note_test.go": "package note\n\nimport \"path/filepath\"\n",
	})
	wrong, read, err := findOSPathImports(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "note", "note.go")
	const reason = ": use cases operate on slash-separated vault paths and domain sentinels, not OS filepath or io/fs"
	want := []string{path + " imports io/fs" + reason, path + " imports path/filepath" + reason}
	if !slices.Equal(wrong, want) {
		t.Errorf("the rule rejects %v, want %v", wrong, want)
	}
	if read != 2 {
		t.Errorf("%d files read, want 2", read)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.RunTests(m))
}
