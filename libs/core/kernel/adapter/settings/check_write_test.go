package settings_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	domainsettings "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const pluginID = "md.numen.sample.speech"

func declaredSettings(owner string) ([]manifest.Setting, bool) {
	switch owner {
	case domainsettings.AppOwner:
		return settings.GetAppDeclaration(), true
	case pluginID:
		return []manifest.Setting{
			{Key: "model", Type: "enum", Title: "Model", Options: manifest.SettingOptions{{Value: "small"}, {Value: "large"}}},
			{Key: "maxTokens", Type: "number", Title: "Tokens"},
			{Key: "temperature", Type: "range", Title: "Temperature", Min: new(float64(0)), Max: new(float64(2))},
			{Key: "label", Type: "string", Title: "Label"},
			{Key: "verbose", Type: "boolean", Title: "Verbose"},
		}, true
	}
	return nil, false
}

func installedThemes(name string) ([]string, bool) {
	if name == "themes" {
		return []string{"preset:numen", "mine:dracula", "plugin:md.numen.sample.themes/night"}, true
	}
	return nil, false
}

func check(path []string, value any) error {
	return settings.CheckWrite(declaredSettings, installedThemes, path, value)
}

func TestAKeyItsOwnerDoesNotDeclareIsRejected(t *testing.T) {
	err := check([]string{pluginID, "mistyped"}, "small")
	if !errors.Is(err, settings.ErrUndeclared) {
		t.Errorf("an undeclared key of a plugin gave %v", err)
	}
}

func TestAKeyOfAnUnknownOwnerIsRejected(t *testing.T) {
	err := check([]string{"md.numen.nobody", "model"}, "small")
	if !errors.Is(err, settings.ErrUndeclared) {
		t.Errorf("a key of an unknown owner gave %v", err)
	}
}

func TestAPluginMayWriteASettingOfTheShell(t *testing.T) {
	if err := check([]string{domainsettings.AppOwner, "appearance", "mode"}, "dark"); err != nil {
		t.Errorf("a declared setting of the shell was rejected: %v", err)
	}
}

func TestAValueOfTheDeclaredTypeIsAccepted(t *testing.T) {
	for what, c := range map[string]struct {
		key   string
		value any
	}{
		"an enum option":   {"model", "small"},
		"a number":         {"maxTokens", 256.0},
		"a range value":    {"temperature", 0.7},
		"a string":         {"label", "draft"},
		"a boolean":        {"verbose", true},
		"the top of range": {"temperature", 2.0},
		"the foot":         {"temperature", 0.0},
	} {
		if err := check([]string{pluginID, c.key}, c.value); err != nil {
			t.Errorf("%s was rejected: %v", what, err)
		}
	}
}

func TestAStringWhereANumberIsDeclaredIsRejected(t *testing.T) {
	err := check([]string{pluginID, "maxTokens"}, "many")
	if !errors.Is(err, settings.ErrWrongType) {
		t.Errorf("a string for a number gave %v", err)
	}
}

func TestANumberWhereABooleanIsDeclaredIsRejected(t *testing.T) {
	err := check([]string{pluginID, "verbose"}, 1.0)
	if !errors.Is(err, settings.ErrWrongType) {
		t.Errorf("a number for a boolean gave %v", err)
	}
}

func TestABooleanWhereAStringIsDeclaredIsRejected(t *testing.T) {
	err := check([]string{pluginID, "label"}, true)
	if !errors.Is(err, settings.ErrWrongType) {
		t.Errorf("a boolean for a string gave %v", err)
	}
}

func TestAValueThatIsNoOptionOfAnEnumIsRejected(t *testing.T) {
	err := check([]string{pluginID, "model"}, "huge")
	if !errors.Is(err, settings.ErrNotAnOption) {
		t.Errorf("a value outside the options gave %v", err)
	}
}

func TestANumberAboveTheTopOfARangeIsRejected(t *testing.T) {
	err := check([]string{pluginID, "temperature"}, 2.1)
	if !errors.Is(err, settings.ErrOutOfRange) {
		t.Errorf("a number above the range gave %v", err)
	}
}

func TestANumberBelowTheFootOfARangeIsRejected(t *testing.T) {
	err := check([]string{pluginID, "temperature"}, -0.1)
	if !errors.Is(err, settings.ErrOutOfRange) {
		t.Errorf("a number below the range gave %v", err)
	}
}

func TestTheScalesOfTheShellTakeNothingOutsideTheirBounds(t *testing.T) {
	for what, c := range map[string]struct {
		key   string
		value float64
		ok    bool
	}{
		"interface above":       {"interfaceScale", 2.01, false},
		"interface below":       {"interfaceScale", 0.79, false},
		"interface at the top":  {"interfaceScale", 2, true},
		"interface at the foot": {"interfaceScale", 0.8, true},
		"text above":            {"textScale", 1.76, false},
		"text below":            {"textScale", 0.79, false},
		"text at the top":       {"textScale", 1.75, true},
		"text at the foot":      {"textScale", 0.8, true},
	} {
		err := check([]string{domainsettings.AppOwner, "appearance", c.key}, c.value)
		if c.ok && err != nil {
			t.Errorf("%s was rejected: %v", what, err)
		}
		if !c.ok && !errors.Is(err, settings.ErrOutOfRange) {
			t.Errorf("%s gave %v", what, err)
		}
	}
}

func TestTheModeOfTheShellIsOneOfTheWordsItDeclares(t *testing.T) {
	err := check([]string{domainsettings.AppOwner, "appearance", "mode"}, "purple")
	if !errors.Is(err, settings.ErrNotAnOption) {
		t.Errorf("a mode outside the options gave %v", err)
	}
}

func TestAThemeThatIsNotInstalledIsRejected(t *testing.T) {
	err := check([]string{domainsettings.AppOwner, "appearance", "theme"}, "mine:absent")
	if !errors.Is(err, settings.ErrNotAnOption) {
		t.Errorf("a theme outside the registry gave %v", err)
	}
}

func TestAThemeOfAPluginIsAcceptedWhileItIsInstalled(t *testing.T) {
	err := check([]string{domainsettings.AppOwner, "appearance", "theme"}, "plugin:md.numen.sample.themes/night")
	if err != nil {
		t.Errorf("an installed plugin theme was rejected: %v", err)
	}
}

func TestTheNamingOfTheShellAcceptsABooleanOnly(t *testing.T) {
	err := check([]string{domainsettings.AppOwner, "naming", "syncTitleAndFilename"}, "yes")
	if !errors.Is(err, settings.ErrWrongType) {
		t.Errorf("a string for the naming gave %v", err)
	}
}
