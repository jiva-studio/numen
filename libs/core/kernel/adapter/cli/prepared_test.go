package cli_test

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func preparing(s search.Search) search.VaultPreparing {
	return func(context.Context, domain.Vault) (search.Search, error) { return s, nil }
}

func preparingRead(r indexing.ReadWholeVault) indexing.VaultPreparing {
	return func(context.Context, domain.Vault) (indexing.ReadWholeVault, error) { return r, nil }
}
