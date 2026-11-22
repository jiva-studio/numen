package container

import (
	windowidentity "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/identity"
	licensingadapter "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/licensing"
	"github.com/jiva-studio/numen/libs/core/kernel/port/identityport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
)

func openIdentityUseCases(install pluginInstall, vaultDeps vaultWiring) *windowidentity.UseCases {
	if install.KeyStore == nil {
		return nil
	}
	deps := identity.NewDeps(install.KeyStore, licensingadapter.NewAdapter(nil), vaultDeps.Clock)
	return &windowidentity.UseCases{
		Get:           identity.NewGetIdentity(deps),
		GenerateWords: identity.NewGenerateRecoveryWords(deps),
		Create:        identity.NewCreateIdentity(deps),
		Delete:        identity.NewDeleteIdentity(deps),
	}
}

// openAccount serves the identity operations of the plugins from the key store the resources hold when a call is made.
func openAccount(res *Resources) identityport.Operations {
	return identity.NewAccount(func() pluginport.KeyStore { return res.KeyStore }, licensingadapter.NewAdapter(nil), GetClock(res))
}
