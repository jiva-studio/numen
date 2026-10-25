package file_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/file"
)

func TestAFolderIsMadeWithTheFoldersAboveIt(t *testing.T) {
	v := testsupport.NewVault(t, nil)

	if err := file.NewCreateFolder(vaultwriter.VaultWriters{}).Execute(t.Context(), v, "a/b/c"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(v.Path, "a", "b", "c")); err != nil || !info.IsDir() {
		t.Fatalf("the folder was not made: %v", err)
	}
}
