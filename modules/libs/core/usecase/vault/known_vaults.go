package vault

import (
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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

// KnownVaults is every vault this installation holds, each with what is true of
// it now.
//
// The list remembers where a vault was last seen and which one was opened last;
// whether the folder is still there is a question for the machine. Every screen
// showing the vaults asks both, and asking them together is what keeps one
// screen from marking a vault another leaves unmarked.
type KnownVaults struct {
	Registry port.VaultRegistry
	Folders  FolderCheck
}

// NewKnownVaults is what the list is read out of and what a vault's folder is
// opened through.
func NewKnownVaults(registry port.VaultRegistry, readers port.VaultReaders) KnownVaults {
	return KnownVaults{Registry: registry, Folders: NewFolderCheck(readers)}
}

// Execute is the list. showing is the vault being worked; the empty identity
// asks the list which vault the next window opens, which is the answer where
// nobody is sitting in front of one.
func (u KnownVaults) Execute(showing domain.VaultID) ([]KnownVault, error) {
	held, err := u.Registry.List()
	if err != nil {
		return nil, err
	}
	if len(held) == 0 {
		return nil, nil
	}
	if showing == "" {
		last, recorded, err := u.Registry.Last()
		if err != nil {
			return nil, err
		}
		if recorded {
			showing = last.ID
		}
	}

	known := make([]KnownVault, 0, len(held))
	for _, v := range held {
		known = append(known, KnownVault{
			Vault:     v,
			IsMissing: u.Folders.Execute(v),
			IsCurrent: showing != "" && v.ID == showing,
		})
	}
	return known, nil
}
