package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// accountRequirements says that the identity requirement holds while this device has an identity or is on the way to one.
type accountRequirements struct{ account *identity.Account }

func (r accountRequirements) IsRequirementMet(ctx context.Context, requirement string) bool {
	return requirement == manifest.RequirementIdentity && r.account.IsPresent(ctx)
}

// requireAccount makes the conditions the manifests require follow the account, and makes the plugins that
// require one start and stop with it.
func requireAccount(management *PluginManagement, account *identity.Account) {
	management.Requirements.SetSource(accountRequirements{account: account})
	account.OnChange(func() { management.Keeper.Refresh(context.Background()) })
}
