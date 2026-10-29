package registry

import (
	"slices"

	"github.com/jiva-studio/numen/libs/core/usecase/vault/check"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// KnownVault is one vault on the list, with what is true of it now.
type KnownVault struct {
	Vault domain.Vault

	// IsMissing is a folder that is absent. The vault stays on the list until the user
	// removes it.
	IsMissing bool

	// IsCurrent is the one vault the rest of a screen is about: the one a window
	// is showing, or the one the next window opens.
	IsCurrent bool
}

// GetKnownVaults returns every vault in this installation, each with its current
// state. The current vault comes first and the rest follow in list order.
//
// The list records where a vault was last seen and which one was opened last;
// whether the folder still exists is determined by the machine. Every screen
// showing vaults checks both, and checking them together keeps one screen from
// marking a vault that another leaves unmarked.
type GetKnownVaults struct {
	registry vaultport.VaultRegistry
	folders  check.FolderCheck
}

// NewGetKnownVaults is what the list is read out of and what a vault's folder is
// opened through.
func NewGetKnownVaults(registry vaultport.VaultRegistry, readers vaultport.VaultReaders) GetKnownVaults {
	return GetKnownVaults{registry: registry, folders: check.NewFolderCheck(readers)}
}

// Execute returns the list. showing is the vault in use; the empty identity
// requests the vault the next window opens, which is the answer where no vault is
// open.
func (u GetKnownVaults) Execute(showing domain.VaultID) ([]KnownVault, error) {
	target, err := u.registry.List()
	if err != nil {
		return nil, err
	}
	if len(target) == 0 {
		return nil, nil
	}
	if showing == "" {
		last, recorded, err := u.registry.GetLastVault()
		if err != nil {
			return nil, err
		}
		if recorded {
			showing = last.ID
		}
	}

	recognized := make([]KnownVault, 0, len(target))
	for _, v := range target {
		one := KnownVault{
			Vault:     v,
			IsMissing: u.folders.Execute(v),
			IsCurrent: showing != "" && v.ID == showing,
		}
		if one.IsCurrent {
			recognized = slices.Insert(recognized, 0, one)
			continue
		}
		recognized = append(recognized, one)
	}
	return recognized, nil
}
