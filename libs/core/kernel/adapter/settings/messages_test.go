package settings_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestTheNoticesOfReadingTheFileNameTheKeyAndWhatHappens(t *testing.T) {
	for body, want := range map[string]string{
		`{"app":{"appearance":{"textScale":9}}}`:   `numen.json: "app.appearance.textScale": 9 is out of range 0.8..1.75; the value is ignored`,
		`{"app":{"appearance":{"tint":"warm"}}}`:   `numen.json: "app.appearance.tint": no such setting; the value is ignored`,
		`{"app":{"ai":{"default":{"agent":"a"}}}}`: `numen.json: "app.ai": no such setting; the value is ignored`,
		`{"app":3}`: `numen.json: app is not a JSON object; its settings are ignored`,
		`{"importing":{"ffmpeg":{"command":[]}}}`:        `numen.json: importing is not a valid plugin id; its settings are ignored`,
		`{"md.numen.sample.speech":3}`:                   `numen.json: md.numen.sample.speech is not a JSON object; its settings are ignored`,
		`{"md.numen.ghost":{"model":"small"}}`:           `numen.json: no installed plugin md.numen.ghost; its settings are ignored`,
		`{"md.numen.sample.speech":{"temperature":5}}`:   `numen.json: "md.numen.sample.speech.temperature": 5 is out of range 0..2; the value is ignored`,
		`{"md.numen.sample.speech":{"model":"huge"}}`:    `numen.json: "md.numen.sample.speech.model": "huge" is not one of the options; the value is ignored`,
		`{"md.numen.sample.speech":{"mistyped":1}}`:      `numen.json: "md.numen.sample.speech.mistyped": no such setting; the value is ignored`,
		`{"app":{"appearance":{"theme":"mine:absent"}}}`: `numen.json: "app.appearance.theme": "mine:absent" is not one of the options listed in themes; the default preset:numen applies`,
	} {
		cfg, err := openAt(write(t, body))
		if err != nil {
			t.Fatalf("%s: the file was rejected: %v", body, err)
		}
		cfg.DropInvalid(declaredSettings, installedThemes)
		if !slices.Contains(cfg.Notices, want) {
			t.Errorf("%s: the notices are %q, want %q among them", body, cfg.Notices, want)
		}
	}
}

func TestTheErrorsOfAWriteNameTheKeyAndTheFault(t *testing.T) {
	for _, one := range []struct {
		path   []string
		value  any
		want   string
		isKind error
	}{
		{[]string{"app"}, 1, `"app": no such setting`, settingsdomain.ErrUndeclared},
		{[]string{"md.numen.nobody", "model"}, "small", `"md.numen.nobody": no plugin declares settings under this name`, settingsdomain.ErrUndeclared},
		{[]string{pluginID, "mistyped"}, "small", `"md.numen.sample.speech.mistyped": no such setting`, settingsdomain.ErrUndeclared},
		{[]string{pluginID, "model", "size"}, "small", `"md.numen.sample.speech.model.size": no such setting: a plugin setting key is a single name`, settingsdomain.ErrUndeclared},
		{[]string{pluginID, "enabled"}, "yes", `"md.numen.sample.speech.enabled": wrong type: expected true or false`, settingsdomain.ErrWrongType},
		{[]string{pluginID, "verbose"}, 1.0, `"md.numen.sample.speech.verbose": wrong type: expected true or false`, settingsdomain.ErrWrongType},
		{[]string{pluginID, "label"}, 1.0, `"md.numen.sample.speech.label": wrong type: expected a string`, settingsdomain.ErrWrongType},
		{[]string{pluginID, "maxTokens"}, "many", `"md.numen.sample.speech.maxTokens": wrong type: expected a number`, settingsdomain.ErrWrongType},
		{[]string{pluginID, "model"}, 3.0, `"md.numen.sample.speech.model": wrong type: expected a string`, settingsdomain.ErrWrongType},
		{[]string{pluginID, "temperature"}, 5.0, `"md.numen.sample.speech.temperature": 5 is out of range 0..2`, settingsdomain.ErrOutOfRange},
		{[]string{pluginID, "model"}, "huge", `"md.numen.sample.speech.model": "huge" is not one of the options`, settingsdomain.ErrNotAnOption},
		{[]string{settingsdomain.AppOwner, "appearance", "theme"}, "mine:absent", `"app.appearance.theme": "mine:absent" is not one of the options listed in themes`, settingsdomain.ErrNotAnOption},
	} {
		err := check(one.path, one.value)
		if err == nil || err.Error() != one.want || !errors.Is(err, one.isKind) {
			t.Errorf("%v = %v: said %v, want %q (%v)", one.path, one.value, err, one.want, one.isKind)
		}
	}
}

func TestFormatRangeWithoutUpperLimitPrintsInf(t *testing.T) {
	declared := func(owner string) ([]manifest.Setting, bool) {
		return []manifest.Setting{{Key: "port", Type: "number", Title: "Port", Min: new(float64(1024))}}, owner == "org.example.relay"
	}
	err := settingsdomain.CheckValue(declared, nil, []string{"org.example.relay", "port"}, 80.0)
	if want := `"org.example.relay.port": 80 is out of range 1024..+inf`; err == nil || err.Error() != want {
		t.Errorf("said %v, want %q", err, want)
	}
}

func TestAScaleOutsideItsBoundsNamesTheSettingAndTheRange(t *testing.T) {
	err := settings.GetTextScaleBounds().Check("app.appearance.textScale", 9)
	if want := `"app.appearance.textScale": 9 is out of range 0.8..1.75`; err == nil || err.Error() != want {
		t.Errorf("said %v, want %q", err, want)
	}
}

func TestNonObjectJSONFileIsRejectedWithTheBytePosition(t *testing.T) {
	for _, one := range []struct{ body, want string }{
		{`null`, "numen.json: the file is not a JSON object"},
		{`{"app":`, "invalid JSON at byte"},
		{`[1]`, "a value has the wrong type at byte"},
	} {
		body, want := one.body, one.want
		_, err := openAt(write(t, body))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: said %v, want %q in it", body, err, want)
		}
	}
	if _, err := openAt(write(t, `null`)); !errors.Is(err, settingsdomain.ErrNotASetting) {
		t.Errorf("null: said %v, want a value that is not a setting", err)
	}
}

func TestAWriteIntoANonObjectValueNamesThePath(t *testing.T) {
	path := write(t, `{"app":{"appearance":3}}`)
	err := settings.NewFiles().Save(path, newDocument(), theme("preset:nord"))
	if want := `"app": "appearance": not a JSON object; a setting cannot be written inside it`; err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("said %v, want %q in it", err, want)
	}
}

func TestAWriteIntoASectionWithARepeatedNameNamesIt(t *testing.T) {
	path := write(t, `{"app":{"appearance":{"theme":"preset:numen","theme":"preset:dracula"}}}`)
	err := settings.NewFiles().Save(path, newDocument(), theme("preset:nord"))
	if want := `"app": "appearance": "theme" appears more than once`; err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("said %v, want %q in it", err, want)
	}
}

func TestWriteSettingWithEmptyPathFails(t *testing.T) {
	path := write(t, `{}`)
	err := settings.NewFiles().Save(path, newDocument(), settings.Setting{At: nil, Written: 1})
	if want := "the setting path is empty"; err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("said %v, want %q in it", err, want)
	}
}
