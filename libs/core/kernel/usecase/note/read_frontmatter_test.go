package note_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// The type and the properties are what the file holds now, so a note written a moment ago has them
// before any index does.
func TestTheFrontmatterIsReadFromTheCurrentFile(t *testing.T) {
	t.Parallel()
	v := vaultfixture.NewVault(t, map[string]string{"Harbor.md": "---\ntype: pdf\nshelf: east\n---\n# Harbor\n"})
	u := note.NewReadFrontmatter(vaultreader.Readers{})

	noteType, properties, err := u.Execute(t.Context(), v, "Harbor.md")
	if err != nil || noteType != "pdf" || properties["shelf"] != "east" {
		t.Fatalf("got %q, %v, %v", noteType, properties, err)
	}

	rewritten := "---\ntype: video\nshelf: west\n---\n# Harbor\n"
	if err := os.WriteFile(filepath.Join(v.Path, "Harbor.md"), []byte(rewritten), 0o644); err != nil {
		t.Fatal(err)
	}
	noteType, properties, err = u.Execute(t.Context(), v, "Harbor.md")
	if err != nil || noteType != "video" || properties["shelf"] != "west" {
		t.Errorf("after the rewrite got %q, %v, %v", noteType, properties, err)
	}
}

// A note with no frontmatter is an ordinary note with an empty set of properties, never none.
func TestANoteWithoutFrontmatterHasNoProperties(t *testing.T) {
	t.Parallel()
	v := vaultfixture.NewVault(t, map[string]string{"Harbor.md": "# Harbor\n"})

	noteType, properties, err := note.NewReadFrontmatter(vaultreader.Readers{}).Execute(t.Context(), v, "Harbor.md")
	if err != nil || noteType != domain.TypeNote || properties == nil || len(properties) != 0 {
		t.Errorf("got %q, %v, %v", noteType, properties, err)
	}
}

// A path that is no note is not read: it has no type and no properties, and a file that is not there
// is no failure.
func TestAFileThatIsNoNoteIsNotRead(t *testing.T) {
	t.Parallel()
	v := vaultfixture.NewVault(t, nil)

	noteType, properties, err := note.NewReadFrontmatter(vaultreader.Readers{}).Execute(t.Context(), v, "Harbor.png")
	if err != nil || noteType != "" || properties == nil || len(properties) != 0 {
		t.Errorf("got %q, %v, %v", noteType, properties, err)
	}
}

// A note that is not there fails with the reader's error.
func TestANoteThatIsNotThereFails(t *testing.T) {
	t.Parallel()
	v := vaultfixture.NewVault(t, nil)

	if _, _, err := note.NewReadFrontmatter(vaultreader.Readers{}).Execute(t.Context(), v, "Harbor.md"); err == nil {
		t.Error("a note that is not there was read")
	}
}
