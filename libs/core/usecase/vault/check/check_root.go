package check

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// CheckRoot rejects a root that is not a suitable notes folder: a path that is
// not absolute, a filesystem root, and the user's home directory.
type CheckRoot struct {
	identity vaultport.VaultIdentity
	home     machineport.HomeFolder
}

func NewCheckRoot(identity vaultport.VaultIdentity, home machineport.HomeFolder) *CheckRoot {
	return &CheckRoot{identity: identity, home: home}
}

// Execute returns nil for a root that may become a vault. A use case that is
// not fully configured rejects every root.
func (u *CheckRoot) Execute(root string) error {
	if u == nil || u.identity == nil || u.home == nil {
		return fmt.Errorf("%s cannot be checked; the check of a vault folder is not configured", root)
	}
	if !domain.IsAbsPath(root) {
		return fmt.Errorf("%s is not an absolute path", root)
	}
	pos := u.getName(root)
	if domain.IsRootPath(pos) {
		return fmt.Errorf("%s is the root of this filesystem; a vault cannot be a filesystem root", pos)
	}
	if home, err := u.home.GetHomeFolder(); err == nil && u.getName(home) == pos {
		return fmt.Errorf("%s is the home directory; a vault cannot be the home directory", pos)
	}
	return nil
}

func (u *CheckRoot) getName(path string) string {
	return u.identity.GetName(domain.CleanPath(path))
}
