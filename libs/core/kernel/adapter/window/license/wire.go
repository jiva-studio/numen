package license

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

func writeLicenses(entries []license.Info) []*v1.LicenseInfo {
	out := make([]*v1.LicenseInfo, 0, len(entries))
	for _, one := range entries {
		out = append(out, writeLicense(one))
	}
	return out
}

func writeLicense(info license.Info) *v1.LicenseInfo {
	contributors := make([]*v1.LicenseContributor, 0, len(info.Contributors))
	for _, c := range info.Contributors {
		contributors = append(contributors, &v1.LicenseContributor{
			PluginId:    c.PluginID,
			PluginName:  c.PluginName,
			Description: c.Description,
		})
	}
	providers := make([]*v1.LicenseProviderOption, 0, len(info.Providers))
	for _, p := range info.Providers {
		providers = append(providers, &v1.LicenseProviderOption{
			Provider:       p.Provider,
			ProviderName:   p.ProviderName,
			ProductId:      p.ProductID,
			CheckoutUrl:    p.CheckoutURL,
			PaymentMethods: p.PaymentMethods,
		})
	}
	return &v1.LicenseInfo{
		Id:             info.ID,
		Title:          info.Title,
		Contributors:   contributors,
		Providers:      providers,
		Status:         writeStatus(info.Status),
		IsActive:       info.IsActive,
		HasKey:         info.HasKey,
		KeyHint:        info.KeyHint,
		ActiveProvider: info.ActiveProvider,
		InstanceName:   info.InstanceName,
		ExpiresUnixMs:  parseUnixMs(info.ExpiresAt),
		VerifiedUnixMs: parseUnixMs(info.VerifiedAt),
	}
}

func writeStatus(status license.Status) v1.LicenseStatus {
	switch status {
	case license.StatusActive:
		return v1.LicenseStatus_LICENSE_STATUS_ACTIVE
	case license.StatusExpired:
		return v1.LicenseStatus_LICENSE_STATUS_EXPIRED
	case license.StatusUnverified, license.StatusDisabled:
		return v1.LicenseStatus_LICENSE_STATUS_UNVERIFIED
	case license.StatusInactive:
		return v1.LicenseStatus_LICENSE_STATUS_INACTIVE
	}
	return v1.LicenseStatus_LICENSE_STATUS_INACTIVE
}

func parseUnixMs(raw string) int64 {
	parsed, ok := license.ParseTime(raw)
	if !ok {
		return 0
	}
	return parsed.UnixMilli()
}
