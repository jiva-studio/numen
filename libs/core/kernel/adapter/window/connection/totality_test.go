package connection

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	connections "github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
)

func TestErrorCodesAreProducedFromFailures(t *testing.T) {
	for want, failure := range map[v1.ErrorCode]error{
		v1.ErrorCode_ERROR_CODE_UNSPECIFIED:      nil,
		v1.ErrorCode_ERROR_CODE_NOT_FOUND:        connection.ErrNotFound,
		v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS:   connections.ErrNameTaken,
		v1.ErrorCode_ERROR_CODE_INVALID:          connections.ErrInvalidField,
		v1.ErrorCode_ERROR_CODE_TYPE_UNAVAILABLE: connection.ErrTypeUnavailable,
		v1.ErrorCode_ERROR_CODE_KEY_STORE_FAILED: connection.ErrKeyStore,
		v1.ErrorCode_ERROR_CODE_CONFLICT:         connection.ErrStaleRevision,
		v1.ErrorCode_ERROR_CODE_UNAVAILABLE:      connections.ErrProviderUnavailable,
	} {
		if got, _ := getErrorCode(failure); got != want {
			t.Errorf("%v produced %v, want %v", failure, got, want)
		}
	}
}
