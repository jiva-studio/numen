package vaults

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestEveryVaultsErrorCodeIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.VaultsErrorCode]error{
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNREADABLE: vaultdomain.ErrUnreadable,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_COPY:       vaultdomain.ErrCopy,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_OVERLAPS:   domain.ErrOverlaps,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_NAME_TAKEN: vaultdomain.ErrNameTaken,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_LAST_VAULT: vaultdomain.ErrLastVault,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_CURRENT:    vaultdomain.ErrShowing,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_NOT_FOUND:  registry.ErrUnknown,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_NO_TRASH:   machine.ErrNoTrash,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSAVED:    window.ErrAnswerPending,
	}, func(err error) v1.VaultsErrorCode {
		reason, _ := getErrorCode(err)
		return reason
	})
}
