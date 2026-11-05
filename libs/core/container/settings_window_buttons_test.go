package container

import "testing"

func hasAppSetting(cfg Config, goos, key string) bool {
	for _, one := range cfg.listAppSettingsOn(goos) {
		if one.Key == key {
			return true
		}
	}
	return false
}

func TestTheWindowButtonsAreListedOnLinuxOnly(t *testing.T) {
	cfg, _ := writeSettings(t, `{}`)

	for goos, want := range map[string]bool{"linux": true, "windows": false, "darwin": false} {
		if got := hasAppSetting(cfg, goos, "window.buttons"); got != want {
			t.Errorf("the window buttons are listed on %s: %v, want %v", goos, got, want)
		}
	}
}

func TestTheOtherSettingsAreListedOnEverySystem(t *testing.T) {
	cfg, _ := writeSettings(t, `{}`)

	for _, goos := range []string{"linux", "windows", "darwin"} {
		if !hasAppSetting(cfg, goos, "appearance.mode") {
			t.Errorf("the mode is not listed on %s", goos)
		}
	}
}
