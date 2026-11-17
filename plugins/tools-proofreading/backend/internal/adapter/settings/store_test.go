package settings

import (
	"testing"

	domainsettings "github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/settings"
)

func TestStoreReturnsDefaultAndUpdatedConfig(t *testing.T) {
	store := NewStore()
	if got := store.GetRunConfig(); got != domainsettings.DefaultConfig() {
		t.Fatalf("initial config = %+v, want %+v", got, domainsettings.DefaultConfig())
	}

	store.Update(map[string]string{
		"batchSize":       "12",
		"overlap":         "3",
		"inFlight":        "2",
		"maxEditDistance": "0.25",
	})
	want := domainsettings.Config{
		BatchSize:       12,
		Overlap:         3,
		InFlight:        2,
		MaxEditDistance: 0.25,
	}
	if got := store.GetRunConfig(); got != want {
		t.Errorf("updated config = %+v, want %+v", got, want)
	}
}
