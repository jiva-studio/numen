package manifestjson_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func TestDecodeReadsLicensesAndProviders(t *testing.T) {
	m, err := manifestjson.Decode([]byte(`{
		"id": "md.example.supporter",
		"name": "Supporter",
		"version": "1.0.0",
		"contributes": {
			"licenses": [
				{
					"id": "supporter",
					"title": "Supporter",
					"description": "Shows the Supporter badge",
					"offlineGraceDays": 45,
					"providers": [
						{"provider": "creem", "productId": "prod_supporter", "url": "https://creem.io/buy/supporter"},
						{"provider": "dodo", "productId": "pdt_supporter"}
					]
				},
				{
					"id": "enterprise",
					"description": "Unlocks enterprise sync"
				}
			]
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}

	grace := 45
	want := []manifest.License{
		{
			ID:               "supporter",
			Title:            "Supporter",
			Description:      "Shows the Supporter badge",
			OfflineGraceDays: &grace,
			Providers: []manifest.LicenseProvider{
				{Provider: "creem", ProductID: "prod_supporter", URL: "https://creem.io/buy/supporter"},
				{Provider: "dodo", ProductID: "pdt_supporter"},
			},
		},
		{
			ID:          "enterprise",
			Description: "Unlocks enterprise sync",
		},
	}
	if !reflect.DeepEqual(m.Contributes.Licenses, want) {
		t.Errorf("licenses = %+v, want %+v", m.Contributes.Licenses, want)
	}
}
