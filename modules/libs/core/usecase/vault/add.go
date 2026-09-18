package vault

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrUnreadable is a path that is not a folder, or a folder this application
// cannot read as the vault it was asked about.
var ErrUnreadable = errors.New("this folder cannot be read as a vault")

// ErrCopy is one identity carried by two folders that are both there. One index
// cannot hold both.
var ErrCopy = errors.New("one vault cannot be in two places")

// Add turns a folder into a vault this installation knows about.
//
// Two things happen, in this order and no other: the folder is given an
// identity that stays with it, and that identity is remembered here. A folder
// is recognised after it moves by the identity it carries.
//
// A name another vault has gets a number appended, and the person renames it
// afterwards.
type Add struct {
	Identity port.VaultIdentity
	Registry port.VaultRegistry
	Now      port.Clock
}

// NewAdd is what turns a folder into a vault: what writes and reads the
// identity the folder carries, the list this installation keeps, and when this
// is happening, which the identity carries.
func NewAdd(identity port.VaultIdentity, registry port.VaultRegistry, now port.Clock) Add {
	return Add{Identity: identity, Registry: registry, Now: now}
}

// Execute takes a path that is already absolute: resolving one against the
// process working directory is something only the caller knows how to do, and a
// use case whose result depends on where it was invoked from is not one.
func (u Add) Execute(root, name string) (domain.Vault, error) {
	if !filepath.IsAbs(root) {
		return domain.Vault{}, fmt.Errorf("vault path must be absolute, got %q", root)
	}
	// One folder has one name here, whichever route reached it, and which
	// routes lead to it is the machine's to say.
	root = u.Identity.GetName(root)
	if err := u.Identity.CheckReadable(root); err != nil {
		return domain.Vault{}, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	held, err := u.Registry.List()
	if err != nil {
		return domain.Vault{}, err
	}
	known := domain.Vaults(held)
	if err := known.CheckFolderFree(root); err != nil {
		return domain.Vault{}, err
	}

	id, err := u.Identity.Ensure(root, u.Now())
	if err != nil {
		return domain.Vault{}, fmt.Errorf("give %s an identity: %w", root, err)
	}

	// This identity may already be registered somewhere else, and that is either
	// a move or a copy. A move is ordinary and the whole reason identity does
	// not depend on the path; a copy cannot be indexed, because two folders
	// claiming one identity would write each other's notes under the same rows.
	//
	// They are told apart by looking: if the old location still carries this
	// identity, both exist and it is a copy.
	existing, found, err := u.Registry.Find(string(id))
	if err != nil {
		return domain.Vault{}, err
	}
	if found && existing.Path != root {
		stillThere, carriesIt, err := u.Identity.GetVaultID(existing.Path)
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
		// The old location is gone, or is no longer this vault: the folder
		// moved, and the registry is what has to catch up.
	}

	v := domain.Vault{ID: id, Name: name, Path: root}
	if v.Name == "" {
		// The folder name is what the user already calls this collection.
		v.Name = filepath.Base(root)
	}
	v.Name = known.FreeName(v.Name, v.ID)
	if err := u.Registry.Save(v); err != nil {
		return domain.Vault{}, err
	}
	return v, nil
}
