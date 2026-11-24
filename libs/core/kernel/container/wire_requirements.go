package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
)

func requireAccount(management *PluginManagement, account *identity.Account) {
	management.Requirements.SetSource(plugin.IdentityRequirement{Presence: account})
	account.OnChange(func() { management.Keeper.Refresh(context.Background()) })
}
