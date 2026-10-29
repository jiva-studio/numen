package vaultreader

import (
	"errors"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Identity gives folders the identity that makes them vaults, and answers
// whether a folder can be read as one at all.
type Identity struct{ Options vaultconfig.Options }

// GetName is the folder with every link on the way to it resolved, which is
// the machine's own name for it however it was reached. A path that resolves
// to nothing — there is no such folder, or a link along it is broken — is
// returned as given, and the caller that opens it reports the error.
func (i Identity) GetName(root string) string {
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return root
	}
	return resolved
}

func (i Identity) CheckReadable(root string) error {
	_, err := Open(root, i.Options)
	return err
}

func (i Identity) EnsureIdentity(root string, timestamp time.Time) (domain.VaultID, error) {
	cfg, err := vaultconfig.Initialize(root, i.Options.ServiceDir, timestamp)
	if err != nil {
		return "", err
	}
	return domain.VaultID(cfg.ID), nil
}

// GetVaultID reads the identity a folder carries without creating one. A folder that is
// gone, or was never a vault, carries none; that is a result, not a failure.
func (i Identity) GetVaultID(root string) (domain.VaultID, bool, error) {
	cfg, err := vaultconfig.ReadConfig(root, i.Options.ServiceDir)
	if errors.Is(err, vaultconfig.ErrNotAVault) {
		return "", false, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return domain.VaultID(cfg.ID), true, nil
}
