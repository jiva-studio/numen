package settings_test

import (
	"errors"
	"testing"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const scansPlugin = "org.example.scans"

func declareScans(owner string) ([]manifest.Setting, bool) {
	if owner != scansPlugin {
		return nil, false
	}
	limit := 10.0
	return []manifest.Setting{
		{Key: "isAutomatic", Type: "boolean"},
		{Key: "depth", Type: "number", Max: &limit},
		{Key: "mode", Type: "enum", Options: manifest.SettingOptions{{Value: "fast"}, {Value: "deep"}}},
		{Key: "theme", Type: "enum", OptionsFrom: settingsdomain.RegistryThemes},
	}, true
}

func listThemes(name string) ([]string, bool) {
	if name != settingsdomain.RegistryThemes {
		return nil, false
	}
	return []string{"preset:night"}, true
}

// A value of the wrong type, out of range or not among the options is refused with the reason of the
// domain rule, and one the declaration takes passes.
func TestCheckSettingsRefusesAValueTheDeclarationDoesNotTake(t *testing.T) {
	cases := map[string]struct {
		key, json string
		want      error
	}{
		"a boolean":              {"isAutomatic", `true`, nil},
		"a text for a boolean":   {"isAutomatic", `"yes"`, settingsdomain.ErrWrongType},
		"a number in range":      {"depth", `4`, nil},
		"a number out of range":  {"depth", `11`, settingsdomain.ErrOutOfRange},
		"an option":              {"mode", `"deep"`, nil},
		"no option":              {"mode", `"slow"`, settingsdomain.ErrNotAnOption},
		"a theme not installed":  {"theme", `"preset:dawn"`, settingsdomain.ErrNotAnOption},
		"an undeclared key":      {"colour", `"red"`, settingsdomain.ErrUndeclared},
		"a number for an option": {"mode", `3`, settingsdomain.ErrWrongType},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			check := settings.NewCheckSettings(jsonCodec{}, declareScans, listThemes, nil)

			err := check.Execute([]settingsdomain.Setting{{Path: []string{scansPlugin, one.key}, JSON: one.json}})

			if one.want == nil {
				if err != nil {
					t.Fatalf("err = %v, want none", err)
				}
				return
			}
			if !errors.Is(err, settingsdomain.ErrNotASetting) || !errors.Is(err, one.want) {
				t.Errorf("err = %v, want ErrNotASetting and %v", err, one.want)
			}
		})
	}
}
