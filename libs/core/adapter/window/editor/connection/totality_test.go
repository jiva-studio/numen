package connection

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	connections "github.com/jiva-studio/numen/libs/core/usecase/connection"
)

func TestErrorCodesAreProducedFromFailures(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.ProviderConnectionErrorCode]error{
		v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_UNSPECIFIED:      nil,
		v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_NOT_FOUND:        connections.ErrNotFound,
		v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_NAME_TAKEN:       connections.ErrNameTaken,
		v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_INVALID_FIELD:    connections.ErrInvalidField,
		v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_TYPE_UNAVAILABLE: connections.ErrTypeUnavailable,
		v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_KEY_STORE_FAILED: connections.ErrKeyStore,
		v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_STALE_REVISION:   connections.ErrStaleRevision,
	}, func(failure error) v1.ProviderConnectionErrorCode {
		code, _ := getErrorCode(failure)
		return code
	})
}
