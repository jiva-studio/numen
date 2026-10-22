package vault_test

import (
	"errors"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

var errRefused = errors.New("the machine failed")

// refusingIdentity is the identity a folder carries, read through the
// filesystem, with a question answered by an error or a fixed answer when a
// test sets one.
type refusingIdentity struct {
	filesystem.VaultIdentity
	checkReadable error
	ensure        error
	// ensureAs, when set, is the identity every folder is given.
	ensureAs domain.VaultID
	// vaultID, when set, answers GetVaultID.
	vaultID func(root string) (domain.VaultID, bool, error)
}

func (r refusingIdentity) CheckReadable(root string) error {
	if r.checkReadable != nil {
		return r.checkReadable
	}
	return r.VaultIdentity.CheckReadable(root)
}

func (r refusingIdentity) EnsureIdentity(root string, at time.Time) (domain.VaultID, error) {
	if r.ensure != nil {
		return "", r.ensure
	}
	if r.ensureAs != "" {
		return r.ensureAs, nil
	}
	return r.VaultIdentity.EnsureIdentity(root, at)
}

func (r refusingIdentity) GetVaultID(root string) (domain.VaultID, bool, error) {
	if r.vaultID != nil {
		return r.vaultID(root)
	}
	return r.VaultIdentity.GetVaultID(root)
}

// refusingRegistry is the list this installation keeps, with each question
// answered by an error when a test sets one.
type refusingRegistry struct {
	*appstate.VaultRegistry
	list, find, save error
}

func (r refusingRegistry) List() ([]domain.Vault, error) {
	if r.list != nil {
		return nil, r.list
	}
	return r.VaultRegistry.List()
}

func (r refusingRegistry) FindVault(nameOrPath string) (domain.Vault, bool, error) {
	if r.find != nil {
		return domain.Vault{}, false, r.find
	}
	return r.VaultRegistry.FindVault(nameOrPath)
}

func (r refusingRegistry) Save(v domain.Vault) error {
	if r.save != nil {
		return r.save
	}
	return r.VaultRegistry.Save(v)
}
