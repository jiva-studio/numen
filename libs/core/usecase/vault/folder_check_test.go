package vault_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// A folder that is there is not missing, and one that has gone is. The list
// remembers where a vault was last seen, so the answer is about the machine and
// not about the row.
func TestAFolderThatHasMissingIsMissing(t *testing.T) {
	t.Parallel()
	check := vaults.NewFolderCheck(vaultreader.Readers{})

	pos := filepath.Join(t.TempDir(), "one")
	if err := os.Mkdir(pos, 0o700); err != nil {
		t.Fatal(err)
	}
	v := domain.Vault{ID: "one", Name: "one", Path: pos}
	if check.Execute(v) {
		t.Error("a folder that is there was called missing")
	}

	if err := os.Remove(pos); err != nil {
		t.Fatal(err)
	}
	if !check.Execute(v) {
		t.Error("a folder that has gone was not called missing")
	}
}

// A path holding a file is not a vault to be read, and is missing like a path
// holding nothing.
func TestAPathContainingAFileIsMissingToo(t *testing.T) {
	t.Parallel()
	pos := filepath.Join(t.TempDir(), "one")
	if err := os.WriteFile(pos, []byte("not a vault"), 0o600); err != nil {
		t.Fatal(err)
	}
	check := vaults.NewFolderCheck(vaultreader.Readers{})
	if !check.Execute(domain.Vault{ID: "one", Name: "one", Path: pos}) {
		t.Error("a file where a vault's folder was is not missing")
	}
}
