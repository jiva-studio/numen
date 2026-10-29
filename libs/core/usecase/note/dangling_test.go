package note_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// A link that a move could not repair now points at nothing. The note holding
// it is never written, so nothing updates it: a move that reports only the
// links it repaired silently leaves the vault broken.

// TestAMoveNamesTheLinksItCouldNotRepair. The note pointing at what moved was
// written in another editor since the vault read it, and its frontmatter no
// longer parses. Its link is left currently, and it is named.
func TestAMoveNamesTheLinksItCouldNotRepair(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"physics/Entropy.md": "A measure of disorder.\n",
		"physics/Heat.md":    "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n---\n# Heat\n",
		"physics/Cold.md":    "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n---\n# Cold\n",
	})

	// The file was modified by another process since the vault read it, and
	// now contains a tab where YAML admits none.
	broken := "---\nlinks:\n\t- to: physics/Entropy.md\n    role: parent\n---\n# Cold\n"
	if err := os.WriteFile(
		filepath.Join(c.vault.Path, filepath.FromSlash("physics/Cold.md")), []byte(broken), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	moved, err := c.move().Execute(t.Context(), c.vault, "physics/Entropy.md", "archive/Thermodynamics.md")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(moved.Repaired, []string{"physics/Heat.md"}) {
		t.Errorf("the links written again are %v", moved.Repaired)
	}
	if !slices.Equal(moved.Dangling, []string{"physics/Cold.md"}) {
		t.Errorf("the links now reaching nothing are %v", moved.Dangling)
	}
	if body := c.read(t, "physics/Cold.md"); body != broken {
		t.Errorf("the note nothing could be written into was written:\n%s", body)
	}
}
