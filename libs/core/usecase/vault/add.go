package vault

import (
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// ErrUnreadable is returned for a path that is not a folder, or a folder this
// application cannot read as the requested vault.
var ErrUnreadable = errors.New("this folder cannot be read as a vault")

// ErrCopy is one identity carried by two folders that are both there. One index
// cannot hold both.
var ErrCopy = errors.New("one vault cannot be in two places")

// Add registers a folder as a vault known to this installation.
//
// Two steps happen, in this order: the folder is given an identity that stays with
// it, and that identity is recorded here. A folder is recognised after it moves by
// the identity it carries.
//
// A name already used by another vault gets a number appended, and the user can
// rename it afterwards.
type Add struct {
	identity vaultport.VaultIdentity
	registry vaultport.VaultRegistry
	now      machineport.Clock
}

// NewAdd is what turns a folder into a vault: what writes and reads the
// identity the folder carries, the list this installation keeps, and when this
// is happening, which the identity carries.
func NewAdd(identity vaultport.VaultIdentity, registry vaultport.VaultRegistry, now machineport.Clock) Add {
	return Add{identity: identity, registry: registry, now: now}
}

// Execute requires an absolute path. Resolving one against the process working
// directory is the caller's responsibility; a use case must not depend on where it
// was invoked from.
func (u Add) Execute(root, name string) (domain.Vault, error) {
	if !domain.IsAbsPath(root) {
		return domain.Vault{}, fmt.Errorf("vault path must be absolute, got %q", root)
	}
	// One folder has one name here, whichever route reached it, and which
	// routes lead to it is the machine's to say.
	root = u.identity.GetName(root)
	if err := u.identity.CheckReadable(root); err != nil {
		return domain.Vault{}, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	target, err := u.registry.List()
	if err != nil {
		return domain.Vault{}, err
	}
	recognized := domain.Vaults(target)
	if err := recognized.CheckOverlap(root); err != nil {
		return domain.Vault{}, err
	}

	id, err := u.identity.EnsureIdentity(root, u.now())
	if err != nil {
		return domain.Vault{}, fmt.Errorf("give %s an identity: %w", root, err)
	}

	// This identity may already be registered elsewhere, as either a move or a copy.
	// A move is ordinary, and is why identity does not depend on the path; a copy
	// cannot be indexed, because two folders claiming one identity would write each
	// other's notes under the same rows.
	//
	// They are distinguished by checking whether the old location still carries this
	// identity: if so, both exist and it is a copy.
	existing, found, err := u.registry.FindVault(string(id))
	if err != nil {
		return domain.Vault{}, err
	}
	if found && existing.Path != root {
		stillThere, carriesIt, err := u.identity.GetVaultID(existing.Path)
		if err != nil {
			return domain.Vault{}, err
		}
		if carriesIt && stillThere == id {
			return domain.Vault{}, fmt.Errorf(
				"%w: %s and %s both carry the identity %s, so they are copies of one "+
					"vault and cannot both be indexed. To make this a separate "+
					"vault, delete its service folder and add it again",
				ErrCopy, root, existing.Path, id)
		}
		// The old location is gone, or is no longer this vault: the folder moved, and the
		// registry must be updated.
	}

	v := domain.Vault{ID: id, Name: name, Path: root}
	if v.Name == "" {
		// The folder name is what the user already calls this collection.
		v.Name = domain.BasePath(root)
	}
	v.Name = recognized.FreeName(v.Name, v.ID)
	if err := u.registry.Save(v); err != nil {
		return domain.Vault{}, err
	}
	return v, nil
}
