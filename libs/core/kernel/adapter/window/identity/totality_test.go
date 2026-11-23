package identity

import (
	"testing"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

func TestIdentityErrorCodesAreProducedFromFailures(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.IdentityErrorCode]error{
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED:       nil,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_WORDS:     identitydomain.ErrInvalidWords,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_IDENTITY_EXISTS:   identitydomain.ErrIdentityExists,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_KEY_STORE_FAILED:  identitydomain.ErrKeyStore,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_SHEETS:    identitydomain.ErrInvalidSheets,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NOT_ENOUGH_SHEETS: identitydomain.ErrNotEnoughSheets,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_OTHER_ACCOUNT:     identitydomain.ErrOtherAccount,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NO_IDENTITY:       identitydomain.ErrNoIdentity,
	}, func(failure error) v1.IdentityErrorCode {
		code, _ := getErrorCode(failure)
		return code
	})
}
