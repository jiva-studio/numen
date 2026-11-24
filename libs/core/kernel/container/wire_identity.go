package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	windowidentity "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/kitpdf"
	licensingadapter "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/licensing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
)

func openIdentityUseCases(install pluginInstall, vaultDeps vaultWiring, dialog editor.PathDialog) *windowidentity.UseCases {
	if install.Keys.Store == nil {
		return nil
	}
	deps := identity.NewDeps(install.Keys.Store, licensingadapter.NewAdapter(nil), vaultDeps.Clock).WithCreateLock(install.Keys.CreateLock)
	return &windowidentity.UseCases{
		Identity: windowidentity.IdentityUseCases{
			Get:           identity.NewGetIdentity(deps),
			GenerateWords: identity.NewGenerateRecoveryWords(deps),
			Create:        identity.NewCreateIdentity(deps),
			Delete:        identity.NewDeleteIdentity(deps),
		},
		Kit: windowidentity.KitUseCases{
			Save:    identity.NewSaveRecoveryKit(deps, openKitSaver(dialog)),
			Restore: identity.NewRestoreIdentityFromKit(deps),
		},
	}
}

func openKitSaver(dialog editor.PathDialog) identity.KitSaver {
	if dialog == nil {
		return kitpdf.Unavailable{}
	}
	return kitpdf.NewSaver(dialog)
}
