package vaults

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func TestEveryErrorCodeIsSavedFromOne(t *testing.T) {
	for want, err := range map[v1.ErrorCode]error{
		v1.ErrorCode_ERROR_CODE_UNREADABLE:     vaultdomain.ErrUnreadable,
		v1.ErrorCode_ERROR_CODE_VAULT_COPY:     vaultdomain.ErrCopy,
		v1.ErrorCode_ERROR_CODE_VAULT_OVERLAPS: domain.ErrOverlaps,
		v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS: vaultdomain.ErrNameTaken,
		v1.ErrorCode_ERROR_CODE_LAST_VAULT:     vaultdomain.ErrLastVault,
		v1.ErrorCode_ERROR_CODE_VAULT_CURRENT:  vaultdomain.ErrShowing,
		v1.ErrorCode_ERROR_CODE_NOT_FOUND:      vaultdomain.ErrUnknown,
		v1.ErrorCode_ERROR_CODE_NO_TRASH:       machine.ErrNoTrash,
		v1.ErrorCode_ERROR_CODE_UNSAVED:        window.ErrAnswerPending,
	} {
		if got, _ := getErrorCode(err); got != want {
			t.Errorf("%v produced %v, want %v", err, got, want)
		}
	}
}
