package vault

import (
	"slices"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// KnownVault is one vault on the list, with what is true of it now.
type KnownVault struct {
	Vault domain.Vault

	// IsMissing is a folder that is not there to be read. The vault stays on the
	// list until somebody forgets it.
	IsMissing bool

	// IsCurrent is the one vault the rest of a screen is about: the one a window
	// is showing, or the one the next window opens.
	IsCurrent bool
}

// GetKnownVaults is every vault this installation holds, each with what is true of
// it now. The current vault comes first and the rest follow as the list holds
// them.
//
// The list remembers where a vault was last seen and which one was opened last;
// whether the folder is still there is a question for the machine. Every screen
// showing the vaults asks both, and asking them together is what keeps one
// screen from marking a vault another leaves unmarked.
type GetKnownVaults struct {
	registry port.VaultRegistry
	folders  FolderCheck
}

// NewGetKnownVaults is what the list is read out of and what a vault's folder is
// opened through.
func NewGetKnownVaults(registry port.VaultRegistry, readers port.VaultReaders) GetKnownVaults {
	return GetKnownVaults{registry: registry, folders: NewFolderCheck(readers)}
}

// Execute is the list. showing is the vault being worked; the empty identity
// asks the list which vault the next window opens, which is the answer where
// nobody is sitting in front of one.
func (u GetKnownVaults) Execute(showing domain.VaultID) ([]KnownVault, error) {
	held, err := u.registry.List()
	if err != nil {
		return nil, err
	}
	if len(held) == 0 {
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

	known := make([]KnownVault, 0, len(held))
	for _, v := range held {
		one := KnownVault{
			Vault:     v,
			IsMissing: u.folders.Execute(v),
			IsCurrent: showing != "" && v.ID == showing,
		}
		if one.IsCurrent {
			known = slices.Insert(known, 0, one)
			continue
		}
		known = append(known, one)
	}
	return known, nil
}
