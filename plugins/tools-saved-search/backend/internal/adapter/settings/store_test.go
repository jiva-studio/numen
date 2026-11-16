package settings_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/internal/adapter/settings"
)

func TestStoreRefreshesAfterTenMinutesByDefault(t *testing.T) {
	if got := settings.NewStore().GetSettings(); got.RefreshMinutes != 10 {
		t.Errorf("settings = %+v, want 10 minutes", got)
	}
}

func TestStoreReadsRefreshMinutesOfZero(t *testing.T) {
	current := settings.NewStore()

	current.Update(map[string]string{"refreshMinutes": "0"})

	if got := current.GetSettings(); got.RefreshMinutes != 0 {
		t.Errorf("settings = %+v, want 0 minutes", got)
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
