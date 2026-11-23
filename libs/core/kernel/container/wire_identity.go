package container

import (
	windowidentity "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/identity"
	licensingadapter "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/licensing"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
)

func openIdentityUseCases(install pluginInstall, vaultDeps vaultWiring, pending *identity.PendingStore) *windowidentity.UseCases {
	if install.KeyStore == nil {
		return nil
	}
	deps := identity.NewDeps(install.KeyStore, licensingadapter.NewAdapter(nil), vaultDeps.Clock).WithPending(pending)
	return &windowidentity.UseCases{
		Identity: windowidentity.IdentityUseCases{
			Get:           identity.NewGetIdentity(deps),
			GenerateWords: identity.NewGenerateRecoveryWords(deps),
			Create:        identity.NewCreateIdentity(deps),
			Delete:        identity.NewDeleteIdentity(deps),
			StartRestore:  identity.NewStartRestore(deps),
		},
		Kit: windowidentity.KitUseCases{
			Create:  identity.NewCreateRecoveryKit(deps),
			Check:   identity.NewCheckRecoveryKit(deps),
			Restore: identity.NewStartRestoreFromKit(deps),
		},
	}
}

func openAccount(res *Resources) *identity.Account {
	return identity.NewAccount(func() pluginport.KeyStore { return res.KeyStore }, licensingadapter.NewAdapter(nil), GetClock(res), res.restore)
}
