package vaultport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// CheckWrite refuses a write to a vault that another plugin manages. The caller is the one vault.WithCaller named.
func CheckWrite(ctx context.Context, v domain.Vault) error {
	return vault.CheckWrite(v.ManagedBy, vault.GetCaller(ctx))
}
