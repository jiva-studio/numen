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
	if !hasNotice(cfg.Notices, "appearance.textScale") {
		t.Errorf("the notices are %q", cfg.Notices)
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
	if !hasNotice(cfg.Notices, "appearance.textScale") || !hasNotice(cfg.Notices, "naming.syncTitleAndFilename") {
		t.Errorf("the notices are %q", cfg.Notices)
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
	if !hasNotice(cfg.Notices, "appearance.tint") {
		t.Errorf("the notices are %q", cfg.Notices)
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
	if len(cfg.Notices) != 0 {
		t.Errorf("the notices are %q", cfg.Notices)
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
	if !hasNotice(cfg.Notices, "appearance.theme") {
		t.Errorf("the notices are %q", cfg.Notices)
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
	if !hasNotice(cfg.Notices, "md.numen.sample.speech", "mistyped") {
		t.Errorf("the notices are %q", cfg.Notices)
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
	if !hasNotice(cfg.Notices, "maxTokens") {
		t.Errorf("the notices are %q", cfg.Notices)
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
	if !hasNotice(cfg.Notices, "model") {
		t.Errorf("the notices are %q", cfg.Notices)
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
	if !hasNotice(cfg.Notices, "temperature") {
		t.Errorf("the notices are %q", cfg.Notices)
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
	if !hasNotice(cfg.Notices, "md.numen.ghost") {
		t.Errorf("the notices are %q", cfg.Notices)
	}
}
