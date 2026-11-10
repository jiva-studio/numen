package file_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// The type is what the file declares now, so a note rewritten a moment ago has its new type before
// the index holds it.
func TestTheNoteTypeIsReadFromTheCurrentFile(t *testing.T) {
	v := testsupport.NewVault(t, map[string]string{"Harbor.md": "---\ntype: pdf\n---\n# Harbor\n"})
	u := file.NewReadNoteType(vaultreader.Readers{})

	if got, err := u.Execute(t.Context(), v, "Harbor.md"); err != nil || got != "pdf" {
		t.Fatalf("got %q, %v", got, err)
	}

	if err := os.WriteFile(filepath.Join(v.Path, "Harbor.md"), []byte("---\ntype: video\n---\n# Harbor\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := u.Execute(t.Context(), v, "Harbor.md"); err != nil || got != "video" {
		t.Errorf("after the rewrite got %q, %v", got, err)
	}
}

// A file that is not a note has no type and is not read, so a file that is not there is no failure.
func TestAFileThatIsNoNoteHasNoType(t *testing.T) {
	v := testsupport.NewVault(t, nil)

	if got, err := file.NewReadNoteType(vaultreader.Readers{}).Execute(t.Context(), v, "Harbor.png"); err != nil || got != "" {
		t.Errorf("got %q, %v", got, err)
	}
}
