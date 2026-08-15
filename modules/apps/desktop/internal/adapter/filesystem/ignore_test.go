package filesystem_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
)

func vaultOf(t *testing.T, files map[string]string, ignore []string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := filesystem.Initialize(root, filesystem.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	if ignore != nil {
		config := filepath.Join(root, filesystem.DefaultServiceDir, "config.json")
		var cfg filesystem.Config
		raw, err := os.ReadFile(config)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			t.Fatal(err)
		}
		cfg.Ignore = ignore
		raw, _ = json.MarshalIndent(cfg, "", "  ")
		if err := os.WriteFile(config, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestAToolsOwnFilesAreNotNotes. An editor leaves a lock beside the file it has
// open, and it answers to the extension of a note.
func TestAToolsOwnFilesAreNotNotes(t *testing.T) {
	root := vaultOf(t, map[string]string{
		"Note.md":               "# Note\n",
		".#Note.md":             "someone@somewhere.1234\n",
		"backup/.git/config.md": "not a note\n",
	}, nil)

	if got := walked(t, root, filesystem.Options{}); !slices.Equal(got, []string{"Note.md"}) {
		t.Errorf("walked %v", got)
	}
}

// TestAVaultSaysWhatToIgnore. What is noise is a property of the vault, so it
// is written in the vault.
func TestAVaultSaysWhatToIgnore(t *testing.T) {
	files := map[string]string{
		"Note.md":            "# Note\n",
		"archive/Old.md":     "# Old\n",
		"archive/deep/Er.md": "# Deeper\n",
		"Draft.tmp.md":       "# Draft\n",
	}

	if got := walked(t, vaultOf(t, files, nil), filesystem.Options{}); len(got) != 4 {
		t.Fatalf("without rules the vault holds %v", got)
	}

	root := vaultOf(t, files, []string{"archive/", "*.tmp.md"})
	if got := walked(t, root, filesystem.Options{}); !slices.Equal(got, []string{"Note.md"}) {
		t.Errorf("walked %v, want only the note that survives both rules", got)
	}
}

// TestIgnoringOneVaultDoesNotIgnoreAnother: the rules travel with the folder
// they were written in.
func TestIgnoringOneVaultDoesNotIgnoreAnother(t *testing.T) {
	files := map[string]string{"Note.md": "# Note\n", "archive/Old.md": "# Old\n"}
	quiet := vaultOf(t, files, []string{"archive/"})
	loud := vaultOf(t, files, nil)

	if got := walked(t, quiet, filesystem.Options{}); len(got) != 1 {
		t.Errorf("the vault with a rule walked %v", got)
	}
	if got := walked(t, loud, filesystem.Options{}); len(got) != 2 {
		t.Errorf("the vault without one walked %v", got)
	}
}
