package container

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

func writeAt(cfg Config, json string, at ...string) error {
	return cfg.GetSettingsWriter()([]settingsport.Setting{{Path: at, JSON: json}})
}

func TestAProfileThatIsNoOptionIsRefusedAndNotSent(t *testing.T) {
	cfg, path := writeSettings(t, `{}`)
	host := &profileHost{}
	cfg.PluginHost = host

	err := writeAt(cfg, `"bogus"`, "app", "performance", "profile")
	if !errors.Is(err, settingsport.ErrNotASetting) {
		t.Fatalf("the write gave %v", err)
	}
	if len(host.profiles) != 0 {
		t.Errorf("profiles sent to the host = %v, want none", host.profiles)
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(raw), "bogus") {
		t.Errorf("the file holds the refused value:\n%s", raw)
	}
}

func TestAScaleOutsideItsBoundsIsRefusedWhoeverWritesIt(t *testing.T) {
	for _, value := range []string{`2.5`, `0.5`, `0`} {
		cfg, path := writeSettings(t, `{}`)
		err := writeAt(cfg, value, "app", "appearance", "interfaceScale")
		if !errors.Is(err, settingsport.ErrNotASetting) {
			t.Errorf("%s gave %v", value, err)
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(raw) != `{}` {
			t.Errorf("%s left the file as:\n%s", value, raw)
		}
	}
}

func TestAStringWhereTheNamingTakesABooleanIsRefused(t *testing.T) {
	cfg, _ := writeSettings(t, `{}`)
	err := writeAt(cfg, `"yes"`, "app", "naming", "syncTitleAndFilename")
	if !errors.Is(err, settingsport.ErrNotASetting) {
		t.Errorf("the write gave %v", err)
	}
}

func TestAThemeNobodyInstalledIsRefused(t *testing.T) {
	cfg, _ := writeSettings(t, `{}`)
	err := writeAt(cfg, `"mine:not-installed"`, "app", "appearance", "theme")
	if !errors.Is(err, settingsport.ErrNotASetting) {
		t.Errorf("the write gave %v", err)
	}
}

func TestAKeyAPluginDoesNotDeclareIsRefused(t *testing.T) {
	cfg, path := writeSettings(t, `{}`)
	err := writeAt(cfg, `"x"`, "md.numen.nobody", "mistyped")
	if !errors.Is(err, settingsport.ErrNotASetting) {
		t.Errorf("the write gave %v", err)
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(raw) != `{}` {
		t.Errorf("the file is now:\n%s", raw)
	}
}

func TestAWriteOfSeveralSettingsIsRefusedWholeWhenOneIsRefused(t *testing.T) {
	cfg, path := writeSettings(t, `{}`)
	err := cfg.GetSettingsWriter()([]settingsport.Setting{
		{Path: []string{"app", "appearance", "mode"}, JSON: `"dark"`},
		{Path: []string{"app", "appearance", "textScale"}, JSON: `9`},
	})
	if !errors.Is(err, settingsport.ErrNotASetting) {
		t.Fatalf("the write gave %v", err)
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(raw), "dark") {
		t.Errorf("the accepted half was written:\n%s", raw)
	}
}
