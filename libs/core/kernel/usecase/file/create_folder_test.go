package file_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

func TestAFolderIsMadeWithTheFoldersAboveIt(t *testing.T) {
	v := vaultfixture.NewVault(t, nil)

	if err := file.NewCreateFolder(vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, file.OpenGate{}).Execute(t.Context(), v, "a/b/c"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(v.Path, "a", "b", "c")); err != nil || !info.IsDir() {
		t.Fatalf("the folder was not made: %v", err)
	}
}
