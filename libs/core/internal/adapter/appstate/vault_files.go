package appstate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// VaultFiles keeps one file to a vault in one folder.
type VaultFiles struct{ dir string }

// VaultFilesAt keeps the files of each vault in dir.
func VaultFilesAt(dir string) *VaultFiles { return &VaultFiles{dir: dir} }

func (f *VaultFiles) Read(_ context.Context, vaultID domain.VaultID) ([]byte, error) {
	pos, err := f.getPath(vaultID)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(pos)
}

// Write replaces the vault's file whole, so a second window writing the same
// vault leaves one of the two and never half of both.
func (f *VaultFiles) Write(_ context.Context, vaultID domain.VaultID, content []byte) error {
	pos, err := f.getPath(vaultID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(pos), privateDirMode); err != nil {
		return err
	}
	return replace(pos, content)
}

// getPath is the file of one vault.
//
// The identity is written into the name, and an identity carrying a separator
func (f *VaultFiles) getPath(vaultID domain.VaultID) (string, error) {
	name := string(vaultID)
	if name == "" || strings.ContainsAny(name, `/\.`) {
		return "", fmt.Errorf("%q is not the identity of a vault", name)
	}
	return filepath.Join(f.dir, name+".json"), nil
}
