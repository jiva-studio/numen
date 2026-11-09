package manifestjson_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func TestAStringSettingMayAskTheHostToGenerateAToken(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(`{
		"id": "org.example.relay", "name": "Relay", "version": "1.0.0",
		"settings": [
			{"key": "secret", "type": "string", "title": "Secret", "generate": "token"},
			{"key": "label", "type": "string", "title": "Label"}
		]
	}`))
	if err != nil {
		t.Fatalf("a string setting with a generated token was rejected: %v", err)
	}
	if got := m.Settings[0].Generate; got != "token" {
		t.Errorf("secret generates %q, want token", got)
	}
	if got := m.Settings[1].Generate; got != "" {
		t.Errorf("a plain setting generates %q, want nothing", got)
	}
	if len(m.Warnings) != 0 {
		t.Errorf("warnings = %q, want none", m.Warnings)
	}
}

func TestTheValidatorRejectsAGeneratedTokenOutsideAPlainStringSetting(t *testing.T) {
	for name, setting := range map[string]string{
		"on a number":             `{"key":"a","type":"number","title":"A","generate":"token"}`,
		"on a boolean":            `{"key":"a","type":"boolean","title":"A","generate":"token"}`,
		"on an enum":              `{"key":"a","type":"enum","title":"A","options":["x"],"generate":"token"}`,
		"together with a default": `{"key":"a","type":"string","title":"A","default":"0123abcd","generate":"token"}`,
		"of an unknown kind":      `{"key":"a","type":"string","title":"A","generate":"uuid"}`,
	} {
		_, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","settings":[` + setting + `]}`))
		if !errors.Is(err, manifest.ErrInvalidSetting) {
			t.Errorf("%s: err = %v, want ErrInvalidSetting", name, err)
		}
	}
}
