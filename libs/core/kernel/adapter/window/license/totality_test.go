package license

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

func TestErrorCodesAreProducedFromFailures(t *testing.T) {
	for want, failure := range map[v1.ErrorCode]error{
		v1.ErrorCode_ERROR_CODE_UNSPECIFIED:              nil,
		v1.ErrorCode_ERROR_CODE_NOT_FOUND:                license.ErrNotFound,
		v1.ErrorCode_ERROR_CODE_INVALID:                  license.ErrInvalidKey,
		v1.ErrorCode_ERROR_CODE_ACTIVATION_LIMIT_REACHED: license.ErrActivationLimitReached,
		v1.ErrorCode_ERROR_CODE_EXPIRED:                  license.ErrExpired,
		v1.ErrorCode_ERROR_CODE_DISABLED:                 license.ErrDisabled,
		v1.ErrorCode_ERROR_CODE_PRODUCT_MISMATCH:         license.ErrProductMismatch,
		v1.ErrorCode_ERROR_CODE_UNAVAILABLE:              license.ErrProviderUnavailable,
		v1.ErrorCode_ERROR_CODE_KEY_STORE_FAILED:         license.ErrKeyStore,
	} {
		if got, _ := getErrorCode(failure); got != want {
			t.Errorf("%v produced %v, want %v", failure, got, want)
		}
	}
}

func TestLicenseStatusesAreProduced(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.LicenseStatus]license.Status{
		v1.LicenseStatus_LICENSE_STATUS_INACTIVE:   license.StatusInactive,
		v1.LicenseStatus_LICENSE_STATUS_ACTIVE:     license.StatusActive,
		v1.LicenseStatus_LICENSE_STATUS_EXPIRED:    license.StatusExpired,
		v1.LicenseStatus_LICENSE_STATUS_UNVERIFIED: license.StatusUnverified,
	}, writeStatus)
}
