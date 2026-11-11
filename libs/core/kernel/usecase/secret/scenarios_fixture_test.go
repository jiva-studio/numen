package secret_test

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
)

// secrets are the use cases over the list of secrets, built the way the container builds them.
type secrets struct {
	Items  items
	Groups groups
	Access access
	List   *secret.List
	Prune  *secret.Prune
}

type items struct {
	Create *secret.Create
	Rename *secret.Rename
	Write  *secret.Write
	Delete *secret.Delete
	Move   *secret.Move
}

type groups struct {
	Rename *secret.RenameGroup
	Delete *secret.DeleteGroup
}

type access struct {
	Allow      *secret.Allow
	Deny       *secret.Deny
	ListStates *secret.ListStates
}

func newSecrets(
	ledger pluginport.SecretLedger, keys pluginport.KeyStore, plugins pluginport.SecretPlugins,
	newID func() (string, error), credentialsPath func(ctx context.Context) string, gate *secret.Gate,
) *secrets {
	base := secret.NewOperation(ledger, keys, plugins, gate)
	return &secrets{
		Items: items{
			Create: secret.NewCreate(base, newID),
			Rename: secret.NewRename(base),
			Write:  secret.NewWrite(base),
			Delete: secret.NewDelete(base),
			Move:   secret.NewMove(base),
		},
		Groups: groups{Rename: secret.NewRenameGroup(base), Delete: secret.NewDeleteGroup(base)},
		Access: access{Allow: secret.NewAllow(base), Deny: secret.NewDeny(base), ListStates: secret.NewListStates(base)},
		List:   secret.NewList(base, credentialsPath),
		Prune:  secret.NewPrune(base),
	}
}
