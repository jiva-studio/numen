package appstate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/atomicfile"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// VaultFiles keeps one file to a vault in one folder.
type VaultFiles struct{ dir string }

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
	return atomicfile.WriteFile(pos, bytes.NewReader(content), privateFileMode)
}

func (f *VaultFiles) getPath(vaultID domain.VaultID) (string, error) {
	name := string(vaultID)
	if name == "" || strings.ContainsAny(name, `/\.`) {
		return "", fmt.Errorf("%q is not the identity of a vault", name)
	}
	return filepath.Join(f.dir, name+".json"), nil
}
