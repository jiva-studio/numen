package vault

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Add turns a folder into a vault this installation knows about.
//
// Two things happen, in this order and no other: the folder is given an
// identity that stays with it, and that identity is remembered here. The order
// matters — a folder recorded in the registry but carrying no identity would be
// unrecognisable the moment it moved.
type Add struct {
	Identity port.VaultIdentity
	Registry port.VaultRegistry
	Now      func() time.Time
}

// Execute takes a path that is already absolute: resolving one against the
// process working directory is something only the caller knows how to do, and a
// use case whose result depends on where it was invoked from is not one.
func (u Add) Execute(root, name string) (domain.Vault, error) {
	if !filepath.IsAbs(root) {
		return domain.Vault{}, fmt.Errorf("vault path must be absolute, got %q", root)
	}
	if err := u.Identity.Readable(root); err != nil {
		return domain.Vault{}, err
	}

	id, err := u.Identity.Ensure(root, u.Now())
	if err != nil {
		return domain.Vault{}, fmt.Errorf("give %s an identity: %w", root, err)
	}

	v := domain.Vault{ID: id, Name: name, Path: root}
	if v.Name == "" {
		// The folder name is what the user already calls this collection.
		v.Name = filepath.Base(root)
	}
	if err := u.Registry.Save(v); err != nil {
		return domain.Vault{}, err
	}
	return v, nil
}
