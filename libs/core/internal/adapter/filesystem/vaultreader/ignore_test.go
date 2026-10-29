package vaultreader_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultfixture"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
)

// TestAToolsOwnFilesAreNotNotes. An editor leaves a lock beside the file it has
// open, and it answers to the extension of a note.
func TestAToolsOwnFilesAreNotNotes(t *testing.T) {
	root := vaultfixture.CreateVault(t, map[string]string{
		"Note.md":               "# Note\n",
		".#Note.md":             "someone@somewhere.1234\n",
		"backup/.git/config.md": "not a note\n",
	}, nil)

	if got := walkWithOptions(t, root, vaultconfig.Options{}); !slices.Equal(got, []string{"Note.md"}) {
		t.Errorf("walked %v", got)
	}
}

// TestAVaultSaysWhatToIgnore. What is noise is a property of the vault, so it
// is written in the vault.
func TestAVaultReturnsWhatToIgnore(t *testing.T) {
	files := map[string]string{
		"Note.md":            "# Note\n",
		"archive/Old.md":     "# Old\n",
		"archive/deep/Er.md": "# Deeper\n",
		"Draft.tmp.md":       "# Draft\n",
	}

	if got := walkWithOptions(t, vaultfixture.CreateVault(t, files, nil), vaultconfig.Options{}); len(got) != 4 {
		t.Fatalf("without rules the vault holds %v", got)
	}

	root := vaultfixture.CreateVault(t, files, []string{"archive/", "*.tmp.md"})
	if got := walkWithOptions(t, root, vaultconfig.Options{}); !slices.Equal(got, []string{"Note.md"}) {
		t.Errorf("walked %v, want only the note that survives both rules", got)
	}
}

// TestAVaultsRulesNarrowAndNeverWiden. A vault's configuration is written by
// the party that synced the folder. It may add paths to leave alone; it may not
// remove the defaults no vault has to request, or a line in a file would expose
// every dotfile folder in the vault to reads and writes.
func TestAVaultsRulesNarrowAndNeverWiden(t *testing.T) {
	root := vaultfixture.CreateVault(t, map[string]string{
		"Note.md":               "# Note\n",
		".#Note.md":             "someone@somewhere.1234\n",
		"backup/.git/config.md": "not a note\n",
	}, []string{"!.*"})

	if got := walkWithOptions(t, root, vaultconfig.Options{}); !slices.Equal(got, []string{"Note.md"}) {
		t.Errorf("walked %v", got)
	}

	writer, err := vaultwriter.OpenForWriting(root, vaultconfig.Options{})
	if err != nil {
		t.Fatal(err)
	}
	// folder let back in is somewhere a program's own file lands.
	err = writer.Import(t.Context(), ".git/hooks/pre-commit", strings.NewReader("#!/bin/sh\n"))
	if !errors.Is(err, vaultconfig.ErrNotANote) {
		t.Errorf("bringing a file into .git answered %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the file landed: %v", err)
	}
}

// A vault's own negation applies to what the same vault excluded, which is what
// a negation is for.
func TestAVaultConsumesBackItsOwnRule(t *testing.T) {
	root := vaultfixture.CreateVault(t, map[string]string{
		"Note.md":      "# Note\n",
		"Old.tmp.md":   "# Old\n",
		"Draft.tmp.md": "# Draft\n",
	}, []string{"*.tmp.md", "!Draft.tmp.md"})

	if got := walkWithOptions(t, root, vaultconfig.Options{}); !slices.Equal(got, []string{"Draft.tmp.md", "Note.md"}) {
		t.Errorf("walked %v", got)
	}
}

// TestIgnoringOneVaultDoesNotIgnoreAnother: the rules travel with the folder
// they were written in.
func TestIgnoringOneVaultDoesNotIgnoreAnother(t *testing.T) {
	files := map[string]string{"Note.md": "# Note\n", "archive/Old.md": "# Old\n"}
	quietVault := vaultfixture.CreateVault(t, files, []string{"archive/"})
	loud := vaultfixture.CreateVault(t, files, nil)

	if got := walkWithOptions(t, quietVault, vaultconfig.Options{}); len(got) != 1 {
		t.Errorf("the vault with a rule walked %v", got)
	}
	if got := walkWithOptions(t, loud, vaultconfig.Options{}); len(got) != 2 {
		t.Errorf("the vault without one walked %v", got)
	}
}
