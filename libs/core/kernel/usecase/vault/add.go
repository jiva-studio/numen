package vault

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Add registers a folder as a vault known to this installation: the folder is
// given an identity that stays with it, then that identity is recorded here.
// A folder is recognized after it moves by the identity it carries.
// A name already used by another vault gets a number appended.
type Add struct {
	identity vaultport.VaultIdentity
	vaults   vaultport.VaultList
	lookup   vaultport.VaultLookup
	saver    vaultport.VaultSaver
	now      machineport.Clock
}

// NewAdd is what turns a folder into a vault: what writes and reads the
// identity the folder carries, the list this installation keeps, and when this
// is happening, which the identity carries.
func NewAdd(
	identity vaultport.VaultIdentity, vaults vaultport.VaultList, lookup vaultport.VaultLookup, saver vaultport.VaultSaver,
	now machineport.Clock,
) Add {
	return Add{identity: identity, vaults: vaults, lookup: lookup, saver: saver, now: now}
}

// Execute requires an absolute path. Resolving one against the process working
// directory is the caller's responsibility; a use case must not depend on where it
// was invoked from.
func (u Add) Execute(root, name string) (domain.Vault, error) {
	if !domain.IsAbsPath(root) {
		return domain.Vault{}, fmt.Errorf("%s is not an absolute path", root)
	}
	// One folder has one name here, whichever route reached it, and which
	// routes lead to it is the machine's to say.
	root = u.identity.GetName(root)
	if err := u.identity.CheckReadable(root); err != nil {
		return domain.Vault{}, domain.NewDetailError(fmt.Sprintf("%s cannot be read as a vault: %v", root, err), vaultdomain.ErrUnreadable, err)
	}

	target, err := u.vaults.List()
	if err != nil {
		return domain.Vault{}, err
	}
	recognized := domain.Vaults(target)
	if err := recognized.CheckOverlap(root); err != nil {
		return domain.Vault{}, err
	}

	id, err := u.identity.EnsureIdentity(root, u.now.Now())
	if err != nil {
		return domain.Vault{}, fmt.Errorf("%s cannot be given a vault identity: %w", root, err)
	}

	// This identity may already be registered elsewhere, as either a move or a copy.
	// A move is ordinary, and is why identity does not depend on the path; a copy
	// cannot be indexed, because two folders claiming one identity would write each
	// other's notes under the same rows.
	//
	// They are distinguished by checking whether the old location still carries this
	// identity: if so, both exist and it is a copy.
	existing, found, err := u.lookup.FindVault(string(id))
	if err != nil {
		return domain.Vault{}, err
	}
	if found && existing.Path != root {
		if err := u.checkNotCopy(id, root, existing.Path); err != nil {
			return domain.Vault{}, err
		}
	}

	v := domain.Vault{ID: id, Name: name, Path: root}
	if v.Name == "" {
		// The folder name is what the user already calls this collection.
		v.Name = domain.BasePath(root)
	}
	v.Name = recognized.FreeName(v.Name, v.ID)
	if err := u.saver.Save(v); err != nil {
		return domain.Vault{}, err
	}
	return v, nil
}

func (u Add) checkNotCopy(id domain.VaultID, root, oldPath string) error {
	stillThere, carriesIt, err := u.identity.GetVaultID(oldPath)
	if err != nil {
		return err
	}
	if carriesIt && stillThere == id {
		return domain.NewDetailError(fmt.Sprintf(
			"%s is a copy of the vault at %s; delete the service folder of %s to add it as a separate vault",
			root, oldPath, root), vaultdomain.ErrCopy)
	}
	return nil
}
