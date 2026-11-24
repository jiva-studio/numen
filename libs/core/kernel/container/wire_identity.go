package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	windowidentity "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/kitpdf"
	licensingadapter "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/licensing"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
)

func openIdentityUseCases(install pluginInstall, vaultDeps vaultWiring, res *Resources, dialog editor.PathDialog) *windowidentity.UseCases {
	if install.Keys.Store == nil || res.Stores.Chains == nil {
		return nil
	}
	deps := identity.NewDeps(install.Keys.Store, licensingadapter.NewAdapter(nil), vaultDeps.Clock).WithProgress(res.progress).WithChains(res.Stores.Chains).WithCreateLock(install.Keys.CreateLock)
	return &windowidentity.UseCases{
		Identity: windowidentity.IdentityUseCases{
			Get:           identity.NewGetIdentity(deps),
			GenerateWords: identity.NewGenerateRecoveryWords(deps),
			Create:        identity.NewCreateIdentity(deps),
			Delete:        identity.NewDeleteIdentity(deps),
			StartRestore:  identity.NewStartRestore(deps),
		},
		Kit: windowidentity.KitUseCases{
			Save:    identity.NewSaveRecoveryKit(deps, openKitSaver(dialog)),
			Restore: identity.NewStartRestoreFromKit(deps),
		},
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
	return identity.NewAccount(func() pluginport.KeyStore { return res.Stores.Keys }, licensingadapter.NewAdapter(nil), GetClock(res), res.progress, res.Stores.Chains)
}

func openKitSaver(dialog editor.PathDialog) identity.KitSaver {
	if dialog == nil {
		return kitpdf.Unavailable{}
	}
	return kitpdf.NewSaver(dialog)
}
