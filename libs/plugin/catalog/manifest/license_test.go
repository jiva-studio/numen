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
		`{"id":"supporter","title":"Supporter","description":"Shows the Supporter badge","offlineGraceDays":14,` +
			`"providers":[{"provider":"creem","productId":"prod_supporter"},{"provider":"dodo","url":"https://checkout.dodopayments.com/buy/pdt_supporter"}]},` +
			`{"id":"custom-pro","description":"Unlocks pro export"}`,
	)
	if err != nil {
		t.Fatal(err)
	}

	got := m.Contributes.Licenses
	if len(got) != 2 {
		t.Fatalf("licenses = %+v, want 2", got)
	}
	if got[0].ID != "supporter" || got[0].Title != "Supporter" || got[0].Description != "Shows the Supporter badge" {
		t.Errorf("first license = %+v", got[0])
	}
	if got[0].OfflineGraceDays == nil || *got[0].OfflineGraceDays != 14 {
		t.Errorf("first license offlineGraceDays = %v, want 14", got[0].OfflineGraceDays)
	}
	wantProviders := []manifest.LicenseProvider{
		{Provider: manifest.LicenseProviderCreem, ProductID: "prod_supporter"},
		{Provider: manifest.LicenseProviderDodo, URL: "https://checkout.dodopayments.com/buy/pdt_supporter"},
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
		"an id that is not kebab-case":        `{"id":"Supporter_Pro","description":"d"}`,
		"an id declared twice":                `{"id":"supporter","description":"a"},{"id":"supporter","description":"b"}`,
		"a title with surrounding space":      `{"id":"supporter","title":" Supporter","description":"d"}`,
		"an empty description":                `{"id":"supporter","description":"   "}`,
		"a description with leading space":    `{"id":"supporter","description":" d"}`,
		"a description with trailing space":   `{"id":"supporter","description":"d "}`,
		"offlineGraceDays below minimum":      `{"id":"supporter","description":"d","offlineGraceDays":0}`,
		"offlineGraceDays above maximum":      `{"id":"supporter","description":"d","offlineGraceDays":366}`,
		"an unknown provider":                 `{"id":"supporter","description":"d","providers":[{"provider":"stripe","productId":"prod_1"}]}`,
		"a provider declared twice":           `{"id":"supporter","description":"d","providers":[{"provider":"creem","productId":"prod_1"},{"provider":"creem","productId":"prod_2"}]}`,
		"a provider without productId or url": `{"id":"supporter","description":"d","providers":[{"provider":"creem"}]}`,
		"a productId with whitespace":         `{"id":"supporter","description":"d","providers":[{"provider":"creem","productId":"prod 1"}]}`,
		"a productId with a slash":            `{"id":"supporter","description":"d","providers":[{"provider":"creem","productId":"prod/1"}]}`,
		"a url that is not https":             `{"id":"supporter","description":"d","providers":[{"provider":"creem","url":"http://creem.io/buy/1"}]}`,
		"a url without a host":                `{"id":"supporter","description":"d","providers":[{"provider":"creem","url":"https:///buy/1"}]}`,
		"a url with whitespace":               `{"id":"supporter","description":"d","providers":[{"provider":"creem","url":"https://creem.io/buy/1 "}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := readLicenses(licenses); !errors.Is(err, manifest.ErrInvalidContribution) {
				t.Errorf("err = %v, want an invalid contribution", err)
			}
		})
	}
}
