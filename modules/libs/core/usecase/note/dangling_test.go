package note_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// A link a move could not repair is a link that now reaches nothing. The note
// holding it is never written, so nothing comes back to it: a move that names
// only what it repaired hands the person a vault it quietly broke.

// TestAMoveNamesTheLinksItCouldNotRepair. The note pointing at what moved was
// written in another editor since the vault read it, and its frontmatter no
// longer parses. Its link is left as it stands, and it is named.
func TestAMoveNamesTheLinksItCouldNotRepair(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"physics/Entropy.md": "A measure of disorder.\n",
		"physics/Heat.md":    "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n---\n# Heat\n",
		"physics/Cold.md":    "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n---\n# Cold\n",
	})

	// Somebody has been in the file since the vault read it, and left a tab
	// where YAML admits none.
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
