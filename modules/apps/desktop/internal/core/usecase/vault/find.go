package vault

import (
	"fmt"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Find resolves what the user typed — a name, a path or an identity — into
// a vault. It is shared by every use case that works on one.
type Find struct {
	Registry port.VaultRegistry
}

func (u Find) Execute(nameOrPath string) (domain.Vault, error) {
	v, found, err := u.Registry.Find(nameOrPath)
	if err != nil {
		return domain.Vault{}, err
	}
	if !found {
		return domain.Vault{}, fmt.Errorf("no vault called %q", nameOrPath)
	}
	return v, nil
}
