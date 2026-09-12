package filesystem

import (
	"errors"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// VaultIdentity gives folders the identity that makes them vaults, and answers
// whether a folder can be read as one at all.
type VaultIdentity struct{ Options Options }

// GetName is the folder with every link on the way to it resolved, which is
// what this machine calls it however it was reached. A path that resolves to
// nothing — there is no such folder, or a link along it is broken — is named as
// it was given, and what is wrong with it is said by whoever opens it.
func (i VaultIdentity) GetName(root string) string {
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		return root
	}
	return real
}

func (i VaultIdentity) Readable(root string) error {
	_, err := Open(root, i.Options)
	return err
}

func (i VaultIdentity) Ensure(root string, at time.Time) (domain.VaultID, error) {
	cfg, err := Initialize(root, i.Options.ServiceDir, at)
	if err != nil {
		return "", err
	}
	return domain.VaultID(cfg.ID), nil
}

// Of reads the identity a folder carries without creating one. A folder that is
// gone, or was never a vault, simply carries none — that is an answer rather
// than a failure.
func (i VaultIdentity) Of(root string) (domain.VaultID, bool, error) {
	cfg, err := ReadConfig(root, i.Options.ServiceDir)
	if errors.Is(err, ErrNotAVault) {
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
