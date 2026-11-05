package settings_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
)

func TestTheWindowButtonsApplyOnLinuxOnly(t *testing.T) {
	for goos, want := range map[string]bool{"linux": true, "windows": false, "darwin": false} {
		if got := settings.IsAvailableOn("window.buttons", goos); got != want {
			t.Errorf("the window buttons on %s: %v, want %v", goos, got, want)
		}
	}
}

func TestEveryOtherSettingAppliesOnEverySystem(t *testing.T) {
	for _, goos := range []string{"linux", "windows", "darwin"} {
		if !settings.IsAvailableOn("appearance.theme", goos) {
			t.Errorf("the theme is not available on %s", goos)
		}
	}
}
