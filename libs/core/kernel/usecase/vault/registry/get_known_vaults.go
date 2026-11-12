package registry

import (
	"slices"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// KnownVault is one vault on the list, with what is true of it now.
type KnownVault struct {
	Vault domain.Vault

	IsMissing bool

	IsCurrent bool
}

// GetKnownVaults returns every vault in this installation, each with its current
// state. The current vault comes first and the rest follow in list order.
// The list records where a vault was last seen and which one was opened last;
// whether the folder still exists is determined by the machine. Checking both
// together keeps one screen from marking a vault that another leaves unmarked.
type GetKnownVaults struct {
	vaults  vaultport.VaultList
	last    vaultport.LastVault
	folders FolderCheck
}

// NewGetKnownVaults is what the list and the last vault are read out of and what a vault's folder is
// opened through.
func NewGetKnownVaults(vaults vaultport.VaultList, last vaultport.LastVault, readers vaultport.VaultReaders) GetKnownVaults {
	return GetKnownVaults{vaults: vaults, last: last, folders: NewFolderCheck(readers)}
}

// Execute returns the list. showing is the vault in use; the empty identity
// requests the vault the next window opens, which is the answer where no vault is
// open.
func (u GetKnownVaults) Execute(showing domain.VaultID) ([]KnownVault, error) {
	target, err := u.vaults.List()
	if err != nil {
		return nil, err
	}
	if len(target) == 0 {
		return nil, nil
	}
	if showing == "" {
		last, recorded, err := u.last.GetLastVault()
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
