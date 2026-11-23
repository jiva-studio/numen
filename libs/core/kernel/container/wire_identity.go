package container

import (
	windowidentity "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/identity"
	licensingadapter "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/licensing"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
)

func openIdentityUseCases(install pluginInstall, vaultDeps vaultWiring, pending *identity.PendingStore) *windowidentity.UseCases {
	if install.KeyStore == nil || install.ChainStore == nil {
		return nil
	}
	deps := identity.NewDeps(install.KeyStore, licensingadapter.NewAdapter(nil), vaultDeps.Clock).
		WithPending(pending).WithChains(install.ChainStore).WithLinks(install.Links)
	return &windowidentity.UseCases{
		Get:           identity.NewGetIdentity(deps),
		GenerateWords: identity.NewGenerateRecoveryWords(deps),
		Create:        identity.NewCreateIdentity(deps),
		Delete:        identity.NewDeleteIdentity(deps),
		StartRestore:  identity.NewStartRestore(deps),
		Link: windowidentity.LinkUseCases{
			Start:  identity.NewStartLink(deps),
			Get:    identity.NewGetLink(deps),
			Decide: identity.NewDecideLink(deps),
			Cancel: identity.NewCancelLink(deps),
		},
		Account: windowidentity.AccountUseCases{
			EnterCode:   identity.NewEnterLinkCode(deps),
			ConfirmJoin: identity.NewConfirmJoin(deps),
			List:        identity.NewListDevices(deps),
			Revoke:      identity.NewRevokeDevice(deps),
		},
	}
}

func openAccount(res *Resources) *identity.Account {
	return identity.NewAccount(func() pluginport.KeyStore { return res.KeyStore }, licensingadapter.NewAdapter(nil), GetClock(res), res.restore, res.ChainStore, res.links)
}
