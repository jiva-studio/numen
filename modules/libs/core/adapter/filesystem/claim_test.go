package filesystem_test

import (
	"os"
	"path/filepath"
	"testing"
)

// A name taken out of the store leaves nothing behind it, the file its claim
// was held on included. A window that only looked at a recording claims its
// name to ask whether it may be edited, and a vault would otherwise fill with
// claims for every recording anybody opened.
func TestANameRemovedLeavesNoClaimBehindIt(t *testing.T) {
	derived, root := store(t)
	const name = "ocr/abc.txt"

	release, err := derived.Claim(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	if err := derived.Write(t.Context(), name, []byte("what was read")); err != nil {
		t.Fatal(err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}

	if err := derived.Remove(t.Context(), name); err != nil {
		t.Fatal(err)
	}

	left, err := os.ReadDir(filepath.Join(root, ".numen", "ocr"))
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range left {
		t.Errorf("%s was left behind", one.Name())
	}
}

// A name nothing ever claimed is removed as it always was.
func TestANameNothingClaimedIsRemoved(t *testing.T) {
	derived, root := store(t)
	const name = "ocr/abc.txt"

	if err := derived.Write(t.Context(), name, []byte("what was read")); err != nil {
		t.Fatal(err)
	}
	if err := derived.Remove(t.Context(), name); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(root, ".numen", name)); !os.IsNotExist(err) {
		t.Errorf("the name is still there: %v", err)
	}
}
