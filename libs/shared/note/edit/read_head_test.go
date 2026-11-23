package edit_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

// The type and the properties are what the file holds now, so a note written a moment ago has them
// before any index does.
func TestReadHead_ReadsTheCurrentFile(t *testing.T) {
	t.Parallel()
	vault := newFiles(map[string]string{"Harbor.md": "---\ntype: pdf\nshelf: east\n---\n# Harbor\n"})

	noteType, properties, err := edit.ReadHead(t.Context(), vault, "Harbor.md")
	if err != nil || noteType != "pdf" || properties["shelf"] != "east" {
		t.Fatalf("got %q, %v, %v", noteType, properties, err)
	}
	vault.put("Harbor.md", "---\ntype: video\nshelf: west\n---\n# Harbor\n")
	noteType, properties, err = edit.ReadHead(t.Context(), vault, "Harbor.md")
	if err != nil || noteType != "video" || properties["shelf"] != "west" {
		t.Errorf("after the rewrite got %q, %v, %v", noteType, properties, err)
	}
}

// A note with no frontmatter is an ordinary note with an empty set of properties, never none.
func TestReadHead_NoteWithoutFrontmatterHasNoProperties(t *testing.T) {
	t.Parallel()
	noteType, properties, err := edit.ReadHead(t.Context(), newFiles(map[string]string{"Harbor.md": "# Harbor\n"}), "Harbor.md")
	if err != nil || noteType != domain.TypeNote || properties == nil || len(properties) != 0 {
		t.Errorf("got %q, %v, %v", noteType, properties, err)
	}
}

// A path that is no note is not read: it has no type and no properties, and a file that is not there
// is no failure.
func TestReadHead_DoesNotReadAFileThatIsNoNote(t *testing.T) {
	t.Parallel()
	noteType, properties, err := edit.ReadHead(t.Context(), newFiles(nil), "Harbor.png")
	if err != nil || noteType != "" || properties == nil || len(properties) != 0 {
		t.Errorf("got %q, %v, %v", noteType, properties, err)
	}
}

// A note that is not there fails with the reader's error.
func TestReadHead_FailsForANoteThatIsNotThere(t *testing.T) {
	t.Parallel()
	if _, _, err := edit.ReadHead(t.Context(), newFiles(nil), "Harbor.md"); err == nil {
		t.Error("a note that is not there was read")
	}
}
