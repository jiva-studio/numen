package registry

import (
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// VaultAdder registers a folder as a vault and returns it.
type VaultAdder interface {
	Execute(root, name string) (domain.Vault, error)
}

// EnsureRequest names the folder to have on the list.
type EnsureRequest struct {
	Root             string
	Name             string
	IsRecordedOpened bool
}

// EnsureRegistered has a folder on the list of vaults: the vault it already is, or the one it is
// added as.
type EnsureRegistered struct {
	identity vaultport.VaultIdentity
	lookup   vaultport.VaultLookup
	adder    VaultAdder
	recorder vaultport.VaultRecorder
}

// NewEnsureRegistered is what a folder is named and looked up by, what adds it where it is not on
// the list, and what records the vault opened last.
func NewEnsureRegistered(
	identity vaultport.VaultIdentity, lookup vaultport.VaultLookup, adder VaultAdder, recorder vaultport.VaultRecorder,
) EnsureRegistered {
	return EnsureRegistered{identity: identity, lookup: lookup, adder: adder, recorder: recorder}
}

// Execute returns the vault the folder is on the list. A vault that is already there is returned as
// it is, and is not recorded as opened.
func (u EnsureRegistered) Execute(request EnsureRequest) (domain.Vault, error) {
	root := u.identity.GetName(request.Root)
	found, isFound, err := u.lookup.FindVault(root)
	if err != nil {
		return domain.Vault{}, err
	}
	if isFound {
		return found, nil
	}
	added, err := u.adder.Execute(root, request.Name)
	if err != nil {
		return domain.Vault{}, err
	}
	if request.IsRecordedOpened {
		if err := u.recorder.RecordOpened(added.ID); err != nil {
			return domain.Vault{}, err
		}
	}
	return added, nil
}
