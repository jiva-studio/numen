package vault_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// A folder that is there is not missing, and one that has gone is. The list
// remembers where a vault was last seen, so the answer is about the machine and
// not about the row.
func TestAFolderThatHasGoneIsMissing(t *testing.T) {
	t.Parallel()
	asking := vaults.NewFolderMissing(filesystem.VaultReaders{})

	at := filepath.Join(t.TempDir(), "one")
	if err := os.Mkdir(at, 0o700); err != nil {
		t.Fatal(err)
	}
	v := domain.Vault{ID: "one", Name: "one", Path: at}
	if asking.Execute(v) {
		t.Error("a folder that is there was called missing")
	}

	if err := os.Remove(at); err != nil {
		t.Fatal(err)
	}
	if !asking.Execute(v) {
		t.Error("a folder that has gone was not called missing")
	}
}

// A path holding a file is not a vault to be read, and is missing like a path
// holding nothing.
func TestAPathHoldingAFileIsMissingToo(t *testing.T) {
	t.Parallel()
	at := filepath.Join(t.TempDir(), "one")
	if err := os.WriteFile(at, []byte("not a vault"), 0o600); err != nil {
		t.Fatal(err)
	}
	asking := vaults.NewFolderMissing(filesystem.VaultReaders{})
	if !asking.Execute(domain.Vault{ID: "one", Name: "one", Path: at}) {
		t.Error("a file where a vault's folder was is not missing")
	}
}
