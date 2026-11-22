package registry

import (
	"github.com/jiva-studio/numen/libs/shared/note/domain"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Find resolves what the user typed — a name, a path or an identity — into
// a vault. It is shared by every use case that works on one.
type Find struct {
	lookup vaultport.VaultLookup
}

// NewFind is what a name, a path or an identity is resolved against: the list
// this installation keeps.
func NewFind(lookup vaultport.VaultLookup) Find {
	return Find{lookup: lookup}
}

func (u Find) Execute(nameOrPath string) (domain.Vault, error) {
	return vaultdomain.FindByName(nameOrPath, u.lookup.FindVault)
}
