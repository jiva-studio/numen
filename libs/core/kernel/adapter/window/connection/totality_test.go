package connection

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	connections "github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
)

func TestErrorCodesAreProducedFromFailures(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.ProviderConnectionErrorCode]error{
		v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_UNSPECIFIED:      nil,
		v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_NOT_FOUND:        connection.ErrNotFound,
		v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_NAME_TAKEN:       connections.ErrNameTaken,
		v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_INVALID_FIELD:    connections.ErrInvalidField,
		v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_TYPE_UNAVAILABLE: connection.ErrTypeUnavailable,
		v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_KEY_STORE_FAILED: connection.ErrKeyStore,
		v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_STALE_REVISION:   connection.ErrStaleRevision,
	}, func(failure error) v1.ProviderConnectionErrorCode {
		code, _ := getErrorCode(failure)
		return code
	})
}
