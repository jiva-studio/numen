package settings_test

import (
	"os"
	"strings"
	"testing"
)

func hasNotice(notices []string, words ...string) bool {
	for _, notice := range notices {
		found := true
		for _, word := range words {
			found = found && strings.Contains(notice, word)
		}
		if found {
			return true
		}
	}
	return false
}

func TestAScaleOutsideItsBoundsIsReadAndIgnoredWithANotice(t *testing.T) {
	cfg, err := openAt(write(t, `{"app":{"appearance":{"textScale":9,"mode":"dark"}}}`))
	if err != nil {
		t.Fatalf("the file was refused: %v", err)
	}
	if got, _ := cfg.GetValue("app", "appearance", "textScale"); got != 1.0 {
		t.Errorf("the text scale in force is %v, want the default", got)
	}
	if got, _ := cfg.GetValue("app", "appearance", "mode"); got != "dark" {
		t.Errorf("the mode in force is %v, want the valid value beside it", got)
	}
	if !hasNotice(cfg.Header.Notices, "appearance.textScale") {
		t.Errorf("the notices are %q", cfg.Header.Notices)
	}
}

func TestAValueOfTheWrongTypeInTheShellIsReadAndIgnoredWithANotice(t *testing.T) {
	cfg, err := openAt(write(t, `{"app":{"appearance":{"textScale":"big"},"naming":{"syncTitleAndFilename":"yes"}}}`))
	if err != nil {
		t.Fatalf("the file was refused: %v", err)
	}
	if got, _ := cfg.GetValue("app", "appearance", "textScale"); got != 1.0 {
		t.Errorf("the text scale in force is %v, want the default", got)
	}
	if got, _ := cfg.GetValue("app", "naming", "syncTitleAndFilename"); got != true {
		t.Errorf("the naming in force is %v, want the default", got)
	}
	if !hasNotice(cfg.Header.Notices, "appearance.textScale") || !hasNotice(cfg.Header.Notices, "naming.syncTitleAndFilename") {
		t.Errorf("the notices are %q", cfg.Header.Notices)
	}
}

func TestAnUndeclaredKeyOfTheAppearanceIsReadAndIgnoredWithANotice(t *testing.T) {
	cfg, err := openAt(write(t, `{"app":{"appearance":{"tint":"warm","mode":"dark"}}}`))
	if err != nil {
		t.Fatalf("the file was refused: %v", err)
	}
	if got, found := cfg.GetValue("app", "appearance", "tint"); found {
		t.Errorf("the undeclared key is in force as %v", got)
	}
	if !hasNotice(cfg.Header.Notices, "appearance.tint") {
		t.Errorf("the notices are %q", cfg.Header.Notices)
	}
}

func TestAFileWithInvalidValuesIsNotRewrittenBecauseOfThem(t *testing.T) {
	const body = `{"app":{"appearance":{"textScale":9,"tint":"warm"}},"md.numen.ghost":{"x":1}}`
	path := write(t, body)
	cfg, err := openAt(path)
	if err != nil {
		t.Fatalf("the file was refused: %v", err)
	}
	cfg.DropInvalid(declaredSettings, installedThemes)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != body {
		t.Errorf("the file is now:\n%s", raw)
	}
}

func TestAValidFileProducesNoNotice(t *testing.T) {
	cfg, err := openAt(write(t, `{"v":1,"app":{"appearance":{"textScale":1.5,"mode":"dark"}},"md.numen.sample.speech":{"model":"small"}}`))
	if err != nil {
		t.Fatal(err)
	}
	cfg.DropInvalid(declaredSettings, installedThemes)
	if len(cfg.Header.Notices) != 0 {
		t.Errorf("the notices are %q", cfg.Header.Notices)
	}
}

func TestAThemeThatIsNotInstalledIsIgnoredWithANotice(t *testing.T) {
	cfg, err := openAt(write(t, `{"app":{"appearance":{"theme":"mine:absent"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	cfg.DropInvalid(declaredSettings, installedThemes)
	if got, _ := cfg.GetValue("app", "appearance", "theme"); got != "preset:numen" {
		t.Errorf("the theme in force is %v, want the default", got)
	}
	if !hasNotice(cfg.Header.Notices, "appearance.theme") {
		t.Errorf("the notices are %q", cfg.Header.Notices)
	}
}

func TestAnUndeclaredKeyOfAPluginIsIgnoredWithANotice(t *testing.T) {
	cfg, err := openAt(write(t, `{"md.numen.sample.speech":{"model":"small","mistyped":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	cfg.DropInvalid(declaredSettings, installedThemes)
	owner := cfg.GetOwner("md.numen.sample.speech")
	if _, found := owner["mistyped"]; found || owner["model"] != "small" {
		t.Errorf("the object of the plugin is %v", owner)
	}
	if !hasNotice(cfg.Header.Notices, "md.numen.sample.speech", "mistyped") {
		t.Errorf("the notices are %q", cfg.Header.Notices)
	}
}

func TestAPluginValueOfTheWrongTypeIsIgnoredWithANotice(t *testing.T) {
	cfg, err := openAt(write(t, `{"md.numen.sample.speech":{"maxTokens":"many","model":"small"}}`))
	if err != nil {
		t.Fatal(err)
	}
	cfg.DropInvalid(declaredSettings, installedThemes)
	owner := cfg.GetOwner("md.numen.sample.speech")
	if _, found := owner["maxTokens"]; found || owner["model"] != "small" {
		t.Errorf("the object of the plugin is %v", owner)
	}
	if !hasNotice(cfg.Header.Notices, "maxTokens") {
		t.Errorf("the notices are %q", cfg.Header.Notices)
	}
}

func TestAPluginValueThatIsNoOptionIsIgnoredWithANotice(t *testing.T) {
	cfg, err := openAt(write(t, `{"md.numen.sample.speech":{"model":"huge"}}`))
	if err != nil {
		t.Fatal(err)
	}
	cfg.DropInvalid(declaredSettings, installedThemes)
	if _, found := cfg.GetOwner("md.numen.sample.speech")["model"]; found {
		t.Error("the model that is no option is in force")
	}
	if !hasNotice(cfg.Header.Notices, "model") {
		t.Errorf("the notices are %q", cfg.Header.Notices)
	}
}

func TestAPluginNumberOutsideItsRangeIsIgnoredWithANotice(t *testing.T) {
	cfg, err := openAt(write(t, `{"md.numen.sample.speech":{"temperature":5}}`))
	if err != nil {
		t.Fatal(err)
	}
	cfg.DropInvalid(declaredSettings, installedThemes)
	if _, found := cfg.GetOwner("md.numen.sample.speech")["temperature"]; found {
		t.Error("the temperature outside its range is in force")
	}
	if !hasNotice(cfg.Header.Notices, "temperature") {
		t.Errorf("the notices are %q", cfg.Header.Notices)
	}
}

func TestAnObjectOfAnUnknownOwnerIsIgnoredWithANotice(t *testing.T) {
	cfg, err := openAt(write(t, `{"md.numen.ghost":{"model":"small"}}`))
	if err != nil {
		t.Fatal(err)
	}
	cfg.DropInvalid(declaredSettings, installedThemes)
	if owner := cfg.GetOwner("md.numen.ghost"); len(owner) != 0 {
		t.Errorf("the object of the unknown owner is in force: %v", owner)
	}
	if !hasNotice(cfg.Header.Notices, "md.numen.ghost") {
		t.Errorf("the notices are %q", cfg.Header.Notices)
	}
}

func TestAnEditorValueOutsideItsBoundsIsReadAndIgnoredWithANoticeThatNamesTheBound(t *testing.T) {
	for _, body := range []string{
		`{"app":{"editor":{"fontSize":33}}}`,
		`{"app":{"editor":{"fontSize":9}}}`,
		`{"app":{"editor":{"lineHeight":1.1}}}`,
		`{"app":{"editor":{"lineHeight":2.5}}}`,
	} {
		cfg, err := openAt(write(t, body))
		if err != nil {
			t.Fatalf("%s: the file was refused: %v", body, err)
		}
		if len(cfg.Header.Notices) != 1 || strings.Contains(cfg.Header.Notices[0], "no such setting") || !strings.Contains(cfg.Header.Notices[0], "editor.") {
			t.Errorf("%s: the notices are %q, want one about the value out of range", body, cfg.Header.Notices)
		}
	}
}

func TestTheDefaultOfAnEditorSettingStaysInForceWhenItsValueIsIgnored(t *testing.T) {
	cfg, err := openAt(write(t, `{"app":{"editor":{"fontSize":99,"lineHeight":"tall","softWrap":"yes","lineNumbers":1,"fontFamily":false}}}`))
	if err != nil {
		t.Fatalf("the file was refused: %v", err)
	}
	cfg.DropInvalid(declaredSettings, installedThemes)
	for key, want := range map[string]any{
		"fontSize":    14.0,
		"lineHeight":  1.6,
		"softWrap":    true,
		"lineNumbers": false,
		"fontFamily":  "",
	} {
		if got, _ := cfg.GetValue("app", "editor", key); got != want {
			t.Errorf("editor.%s in force is %#v, want the default %#v", key, got, want)
		}
	}
	for _, key := range []string{"fontSize", "lineHeight", "softWrap", "lineNumbers", "fontFamily"} {
		if !hasNotice(cfg.Header.Notices, "editor."+key) || hasNotice(cfg.Header.Notices, "editor."+key, "no such setting") {
			t.Errorf("editor.%s has no notice about its value: %q", key, cfg.Header.Notices)
		}
	}
}

func TestAValidFileOfTheEditorGroupProducesNoNotice(t *testing.T) {
	cfg, err := openAt(write(t, `{"v":1,"app":{"editor":{"fontSize":20,"fontFamily":"serif","lineHeight":1.8,"lineNumbers":true,"softWrap":false}}}`))
	if err != nil {
		t.Fatal(err)
	}
	cfg.DropInvalid(declaredSettings, installedThemes)
	if len(cfg.Header.Notices) != 0 {
		t.Errorf("the notices are %q", cfg.Header.Notices)
	}
}
