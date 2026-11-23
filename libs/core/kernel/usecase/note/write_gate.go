package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func writeUnder[T any](
	ctx context.Context, v domain.Vault, admission vaultport.WriteAdmission, write func() (T, error),
) (T, error) {
	var none T
	if err := vaultport.CheckWrite(ctx, v); err != nil {
		return none, err
	}
	release, err := admission.Begin()
	if err != nil {
		return none, err
	}
	defer release()
	return write()
}
