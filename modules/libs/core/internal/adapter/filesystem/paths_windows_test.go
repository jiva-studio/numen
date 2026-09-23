package filesystem

import (
	"path/filepath"
	"testing"
)

// Windows names where a path starts from in three ways, and a path a vault
// holds uses none of them. A backslash and a volume are ordinary characters in
// a name elsewhere, so the rule is written here.
func TestAPathThatNamesWhereItStartsFromIsRefused(t *testing.T) {
	root := t.TempDir()
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		`\Windows\System32\config\SAM`,
		`\\backup\vaults\notes\a.md`,
		`C:notes\a.md`,
		`C:\Windows\System32\config\SAM`,
	} {
		if _, _, err := resolveVaultPath(real, path, DefaultServiceDir); err == nil {
			t.Errorf("resolveVaultPath(%q) was accepted", path)
		}
		if _, _, err := service(real, path, DefaultServiceDir); err == nil {
			t.Errorf("service(%q) was accepted", path)
		}
	}
}
