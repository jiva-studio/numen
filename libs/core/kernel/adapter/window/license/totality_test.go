package license

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

func TestLicenseErrorCodesAreProducedFromFailures(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.LicenseErrorCode]error{
		v1.LicenseErrorCode_LICENSE_ERROR_CODE_UNSPECIFIED:              nil,
		v1.LicenseErrorCode_LICENSE_ERROR_CODE_NOT_FOUND:                license.ErrNotFound,
		v1.LicenseErrorCode_LICENSE_ERROR_CODE_INVALID_KEY:              license.ErrInvalidKey,
		v1.LicenseErrorCode_LICENSE_ERROR_CODE_ACTIVATION_LIMIT_REACHED: license.ErrActivationLimitReached,
		v1.LicenseErrorCode_LICENSE_ERROR_CODE_EXPIRED:                  license.ErrExpired,
		v1.LicenseErrorCode_LICENSE_ERROR_CODE_DISABLED:                 license.ErrDisabled,
		v1.LicenseErrorCode_LICENSE_ERROR_CODE_PRODUCT_MISMATCH:         license.ErrProductMismatch,
		v1.LicenseErrorCode_LICENSE_ERROR_CODE_PROVIDER_UNAVAILABLE:     license.ErrProviderUnavailable,
		v1.LicenseErrorCode_LICENSE_ERROR_CODE_KEY_STORE_FAILED:         license.ErrKeyStore,
	}, func(failure error) v1.LicenseErrorCode {
		code, _ := getErrorCode(failure)
		return code
	})
}

func TestLicenseStatusesAreProduced(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.LicenseStatus]license.Status{
		v1.LicenseStatus_LICENSE_STATUS_INACTIVE:   license.StatusInactive,
		v1.LicenseStatus_LICENSE_STATUS_ACTIVE:     license.StatusActive,
		v1.LicenseStatus_LICENSE_STATUS_EXPIRED:    license.StatusExpired,
		v1.LicenseStatus_LICENSE_STATUS_UNVERIFIED: license.StatusUnverified,
	}, writeStatus)
}
