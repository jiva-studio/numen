package identity

import (
	"testing"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

func TestIdentityErrorCodesAreProducedFromFailures(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.IdentityErrorCode]error{
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED:             nil,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_WORDS:           identitydomain.ErrInvalidWords,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_IDENTITY_EXISTS:         identitydomain.ErrIdentityExists,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_KEY_STORE_FAILED:        identitydomain.ErrKeyStore,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_SHEETS:          identitydomain.ErrInvalidSheets,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NOT_ENOUGH_SHEETS:       identitydomain.ErrNotEnoughSheets,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_OTHER_ACCOUNT:           identitydomain.ErrOtherAccount,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NO_IDENTITY:             identitydomain.ErrNoIdentity,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_LINK_ACTIVE:             identitydomain.ErrLinkActive,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NO_LINK:                 identitydomain.ErrNoLink,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_LINK_TEXT_INVALID:       identitylib.ErrLinkPayload,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_LAST_DEVICE:             identitydomain.ErrLastDevice,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_REVOKE_RACED:            identitydomain.ErrRevokeRaced,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NOT_ALLOWED_THIS_DEVICE: identitylib.ErrSelfRevoke,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_DEVICE_REVOKED:          identitydomain.ErrDeviceRevoked,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNKNOWN_DEVICE:          identitylib.ErrUnknownDevice,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_CHAIN_MARK:      identitydomain.ErrInvalidChainMark,
		v1.IdentityErrorCode_IDENTITY_ERROR_CODE_CHAIN_OUTDATED:          identitydomain.ErrChainOutdated,
	}, func(failure error) v1.IdentityErrorCode {
		code, _ := getErrorCode(failure)
		return code
	})
}
