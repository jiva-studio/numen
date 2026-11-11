//go:build numen_debug

package debugserve

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

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
	_, err = cfg.EnsureVault(abs, filepath.Base(abs), true)
	return err
}
