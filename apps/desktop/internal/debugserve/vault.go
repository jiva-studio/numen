//go:build numen_debug

package debugserve

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// testVaultID is the identity a test run registers its vault under.
const testVaultID = domain.VaultID("01E2ETESTVAULT000000000000")

// RegisterVault puts the folder at vaultPath in the installation's list of vaults, where it is a
// folder the list does not hold yet. A path that names no folder registers nothing.
func RegisterVault(cfg container.Config, vaultPath string) error {
	if vaultPath == "" {
		return nil
	}
	abs, err := filepath.Abs(vaultPath)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}
	reg, err := cfg.Registry()
	if err != nil {
		return err
	}
	_, found, err := reg.FindVault(abs)
	if err != nil || found {
		return err
	}
	v := domain.Vault{ID: testVaultID, Name: filepath.Base(abs), Path: abs}
	if err := reg.Save(v); err != nil {
		return err
	}
	return reg.RecordOpened(v.ID)
}
