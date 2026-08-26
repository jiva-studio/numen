package container_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/settings"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
)

// A configuration nobody has written a naming section into does what an
// installation nobody has configured does. A caller that leaves the section out
// leaves out nothing.
func TestAConfigurationNamingNothingKeepsTheTwoOneName(t *testing.T) {
	if !(container.Config{}).Sync() {
		t.Error("a title and a filename are told apart")
	}
	if !(container.Config{Naming: settings.Defaults().Naming}).Sync() {
		t.Error("the defaults tell a title and a filename apart")
	}
}

// The section a person wrote is the one the window is built from.
func TestAConfigurationCarriesWhatTheSectionSays(t *testing.T) {
	for name, said := range map[string]bool{"one name": true, "told apart": false} {
		t.Run(name, func(t *testing.T) {
			cfg := container.Config{Naming: settings.Naming{SyncTitleAndFilename: &said}}
			if bool(cfg.Sync()) != said {
				t.Errorf("the section says %v and the configuration says %v", said, cfg.Sync())
			}
		})
	}
}
