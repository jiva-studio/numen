package settings_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/internal/adapter/settings"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/usecase/preview"
)

func TestStoreRefreshesAfterTenMinutesByDefault(t *testing.T) {
	current := settings.NewStore()

	if got := current.GetSettings(); got.RefreshMinutes != 10 || got.Preview != preview.DefaultConfig() {
		t.Errorf("settings = %+v, want 10 minutes and the default numbers", got)
	}
}

func TestStoreReadsRefreshMinutesOfZero(t *testing.T) {
	current := settings.NewStore()

	current.Update(map[string]string{"refreshMinutes": "0", "mergeGapBytes": "50"})

	got := current.GetSettings()
	if got.RefreshMinutes != 0 || got.Preview.MergeGapBytes != 50 {
		t.Errorf("settings = %+v, want 0 minutes and a merge gap of 50", got)
	}
	if current.GetConfig() != got.Preview {
		t.Errorf("GetConfig = %+v, want the numbers of GetSettings", current.GetConfig())
	}
}

func TestStoreKeepsTheDefaultForANegativeRefresh(t *testing.T) {
	for _, value := range []string{"-1", "soon", "2.5"} {
		t.Run(value, func(t *testing.T) {
			current := settings.NewStore()

			current.Update(map[string]string{"refreshMinutes": value})

			if got := current.GetSettings().RefreshMinutes; got != 10 {
				t.Errorf("refresh minutes = %d, want the default 10", got)
			}
		})
	}
}

func TestStoreDefaultFolderIsTheVaultFolder(t *testing.T) {
	if got := settings.NewStore().GetDefaultFolder(); got != "" {
		t.Errorf("default folder = %q, want the vault folder", got)
	}
}

func TestStoreReadsTheDefaultFolder(t *testing.T) {
	current := settings.NewStore()

	current.Update(map[string]string{"defaultFolder": "Searches/Harbor"})

	if got := current.GetDefaultFolder(); got != "Searches/Harbor" {
		t.Errorf("default folder = %q, want Searches/Harbor", got)
	}
}
