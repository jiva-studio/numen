package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func readLicenses(licenses string) (manifest.Manifest, error) {
	return manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","contributes":{"licenses":[` + licenses + `]}}`))
}

func TestAManifestDeclaresLicenses(t *testing.T) {
	m, err := readLicenses(
		`{"id":"sync","title":"Numen Sync","description":"Synchronizes your vaults across devices","offlineGraceDays":14,` +
			`"providers":[{"provider":"creem","productId":"prod_sync"},{"provider":"dodo","url":"https://checkout.dodopayments.com/buy/pdt_sync"}]},` +
			`{"id":"custom-pro","description":"Unlocks pro export"}`,
	)
	if err != nil {
		t.Fatal(err)
	}

	got := m.Contributes.Licenses
	if len(got) != 2 {
		t.Fatalf("licenses = %+v, want 2", got)
	}
	if got[0].ID != "sync" || got[0].Title != "Numen Sync" || got[0].Description != "Synchronizes your vaults across devices" {
		t.Errorf("first license = %+v", got[0])
	}
	if got[0].OfflineGraceDays == nil || *got[0].OfflineGraceDays != 14 {
		t.Errorf("first license offlineGraceDays = %v, want 14", got[0].OfflineGraceDays)
	}
	wantProviders := []manifest.LicenseProvider{
		{Provider: manifest.LicenseProviderCreem, ProductID: "prod_sync"},
		{Provider: manifest.LicenseProviderDodo, URL: "https://checkout.dodopayments.com/buy/pdt_sync"},
	}
	if len(got[0].Providers) != 2 || got[0].Providers[0] != wantProviders[0] || got[0].Providers[1] != wantProviders[1] {
		t.Errorf("first license providers = %+v, want %+v", got[0].Providers, wantProviders)
	}
	if got[1].ID != "custom-pro" || got[1].Title != "" || got[1].OfflineGraceDays != nil || len(got[1].Providers) != 0 {
		t.Errorf("second license = %+v", got[1])
	}
}

func TestALicenseIsRejectedWhereItIsWrong(t *testing.T) {
	for name, licenses := range map[string]string{
		"an empty id":                         `{"id":"","description":"d"}`,
		"an id that is not kebab-case":        `{"id":"Sync_Pro","description":"d"}`,
		"an id declared twice":                `{"id":"sync","description":"a"},{"id":"sync","description":"b"}`,
		"a title with surrounding space":      `{"id":"sync","title":" Numen Sync","description":"d"}`,
		"an empty description":                `{"id":"sync","description":"   "}`,
		"a description with leading space":    `{"id":"sync","description":" d"}`,
		"a description with trailing space":   `{"id":"sync","description":"d "}`,
		"offlineGraceDays below minimum":      `{"id":"sync","description":"d","offlineGraceDays":0}`,
		"offlineGraceDays above maximum":      `{"id":"sync","description":"d","offlineGraceDays":366}`,
		"an unknown provider":                 `{"id":"sync","description":"d","providers":[{"provider":"stripe","productId":"prod_1"}]}`,
		"a provider declared twice":           `{"id":"sync","description":"d","providers":[{"provider":"creem","productId":"prod_1"},{"provider":"creem","productId":"prod_2"}]}`,
		"a provider without productId or url": `{"id":"sync","description":"d","providers":[{"provider":"creem"}]}`,
		"a productId with whitespace":         `{"id":"sync","description":"d","providers":[{"provider":"creem","productId":"prod 1"}]}`,
		"a productId with a slash":            `{"id":"sync","description":"d","providers":[{"provider":"creem","productId":"prod/1"}]}`,
		"a url that is not https":             `{"id":"sync","description":"d","providers":[{"provider":"creem","url":"http://creem.io/buy/1"}]}`,
		"a url without a host":                `{"id":"sync","description":"d","providers":[{"provider":"creem","url":"https:///buy/1"}]}`,
		"a url with whitespace":               `{"id":"sync","description":"d","providers":[{"provider":"creem","url":"https://creem.io/buy/1 "}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := readLicenses(licenses); !errors.Is(err, manifest.ErrInvalidContribution) {
				t.Errorf("err = %v, want an invalid contribution", err)
			}
		})
	}
}
