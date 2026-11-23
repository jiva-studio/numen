package container

import (
	windowidentity "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/identity"
	licensingadapter "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/licensing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
)

func openIdentityUseCases(install pluginInstall, vaultDeps vaultWiring) *windowidentity.UseCases {
	if install.KeyStore == nil {
		return nil
	}
	deps := identity.NewDeps(install.KeyStore, licensingadapter.NewAdapter(nil), vaultDeps.Clock).WithCreateLock(install.IdentityLock)
	return &windowidentity.UseCases{
		Get:           identity.NewGetIdentity(deps),
		GenerateWords: identity.NewGenerateRecoveryWords(deps),
		Create:        identity.NewCreateIdentity(deps),
		Delete:        identity.NewDeleteIdentity(deps),
	}
}
