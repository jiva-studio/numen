package manifestjson_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func TestDecodeReadsLicensesAndProviders(t *testing.T) {
	m, err := manifestjson.Decode([]byte(`{
		"id": "md.example.sync",
		"name": "Numen Sync",
		"version": "1.0.0",
		"contributes": {
			"licenses": [
				{
					"id": "sync",
					"title": "Numen Sync",
					"description": "Synchronizes your vaults across devices",
					"offlineGraceDays": 45,
					"providers": [
						{"provider": "creem", "productId": "prod_sync", "url": "https://creem.io/buy/sync"},
						{"provider": "dodo", "productId": "pdt_sync"}
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
			ID:               "sync",
			Title:            "Numen Sync",
			Description:      "Synchronizes your vaults across devices",
			OfflineGraceDays: &grace,
			Providers: []manifest.LicenseProvider{
				{Provider: "creem", ProductID: "prod_sync", URL: "https://creem.io/buy/sync"},
				{Provider: "dodo", ProductID: "pdt_sync"},
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
