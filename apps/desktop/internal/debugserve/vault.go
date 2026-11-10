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
	reg, err := cfg.Registry()
	if err != nil {
		return err
	}
	_, found, err := reg.Lookup.FindVault(abs)
	if err != nil || found {
		return err
	}
	v, err := cfg.AddVault(reg, abs, filepath.Base(abs))
	if err != nil {
		return err
	}
	return reg.Recorder.RecordOpened(v.ID)
}
