package identity

import (
	"testing"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

func TestGetErrorCodeOfIdentityFailures(t *testing.T) {
	for want, failure := range map[v1.ErrorCode]error{
		v1.ErrorCode_ERROR_CODE_UNSPECIFIED:             nil,
		v1.ErrorCode_ERROR_CODE_INVALID_WORDS:           identitydomain.ErrInvalidWords,
		v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS:          identitydomain.ErrIdentityExists,
		v1.ErrorCode_ERROR_CODE_KEY_STORE_FAILED:        identitydomain.ErrKeyStore,
		v1.ErrorCode_ERROR_CODE_INVALID_SHEETS:          identitydomain.ErrInvalidSheets,
		v1.ErrorCode_ERROR_CODE_NOT_ENOUGH_SHEETS:       identitydomain.ErrNotEnoughSheets,
		v1.ErrorCode_ERROR_CODE_OTHER_ACCOUNT:           identitydomain.ErrOtherAccount,
		v1.ErrorCode_ERROR_CODE_NO_IDENTITY:             identitydomain.ErrNoIdentity,
		v1.ErrorCode_ERROR_CODE_LINK_ACTIVE:             identitydomain.ErrLinkActive,
		v1.ErrorCode_ERROR_CODE_NO_LINK:                 identitydomain.ErrNoLink,
		v1.ErrorCode_ERROR_CODE_LINK_TEXT_INVALID:       identitydomain.ErrLinkText,
		v1.ErrorCode_ERROR_CODE_LAST_DEVICE:             identitydomain.ErrLastDevice,
		v1.ErrorCode_ERROR_CODE_REVOKE_RACED:            identitydomain.ErrRevokeRaced,
		v1.ErrorCode_ERROR_CODE_NOT_ALLOWED_THIS_DEVICE: identitydomain.ErrOwnDevice,
		v1.ErrorCode_ERROR_CODE_DEVICE_REVOKED:          identitydomain.ErrDeviceRevoked,
		v1.ErrorCode_ERROR_CODE_NOT_FOUND:               identitydomain.ErrUnknownDevice,
		v1.ErrorCode_ERROR_CODE_INVALID:                 identitydomain.ErrInvalidChainMark,
		v1.ErrorCode_ERROR_CODE_CHAIN_OUTDATED:          identitydomain.ErrChainOutdated,
	} {
		if code, _ := getErrorCode(failure); code != want {
			t.Errorf("%v is written as %v, want %v", failure, code, want)
		}
	}
}
