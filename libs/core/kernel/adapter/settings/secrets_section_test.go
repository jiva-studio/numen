package settings_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const validSecretsSection = `"secrets":[{"id":"s0a1b2c3d","name":"Default","group":"openrouter",` +
	`"allowed":[{"plugin":"md.numen.sample.speech","name":"apiKey","group":"openrouter"}]}]`

func TestSecretsIsNotAPluginId(t *testing.T) {
	if manifest.IsValidID("secrets") {
		t.Error("secrets is a valid plugin id, so the section would clash with a plugin")
	}
}

func TestNoOwnerDeclaresTheSecretsSection(t *testing.T) {
	for _, path := range [][]string{
		{"secrets"},
		{"secrets", "0", "allowed"},
		{"secrets", "s0a1b2c3d", "name"},
	} {
		if err := check(path, "x"); !errors.Is(err, settings.ErrUndeclared) {
			t.Errorf("a write at %v gave %v, want ErrUndeclared", path, err)
		}
	}
}

func TestAMalformedSecretsSectionIsReadWithoutANoticeThatItIsNotAPluginId(t *testing.T) {
	for what, body := range map[string]string{
		"a string":                  `{"secrets":"oops"}`,
		"a list of numbers":         `{"secrets":[1,2]}`,
		"an object":                 `{"secrets":{"a":1}}`,
		"a null":                    `{"secrets":null}`,
		"entries of the wrong type": `{"secrets":[{"id":7,"name":[],"group":{},"allowed":"all"}]}`,
	} {
		t.Run(what, func(t *testing.T) {
			cfg, err := openAt(write(t, body))
			if err != nil {
				t.Fatalf("the file was refused: %v", err)
			}
			if hasNotice(cfg.Notices, "secrets", "plugin id") {
				t.Errorf("the notices are %q", cfg.Notices)
			}
		})
	}
}

func TestAValidSecretsSectionIsReadWithoutANotice(t *testing.T) {
	cfg, err := openAt(write(t, `{`+validSecretsSection+`}`))
	if err != nil {
		t.Fatal(err)
	}
	if hasNotice(cfg.Notices, "secrets") {
		t.Errorf("the notices are %q", cfg.Notices)
	}
}

func TestTheSecretsSectionIsNotThePrivateSettingsOfAPlugin(t *testing.T) {
	cfg, err := openAt(write(t, `{`+validSecretsSection+`}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, found := cfg.GetValue("secrets", "0", "name"); found {
		t.Error("the secrets section is read as the settings of a plugin named secrets")
	}
}

func TestAWriteOfAnotherSettingKeepsTheSecretsSectionAsItWas(t *testing.T) {
	path := write(t, `{`+validSecretsSection+`}`)
	cfg := container.NewConfig(container.Config{
		Paths: container.Paths{SettingsPath: path},
	})

	err := container.GetSettingsWriter(cfg)([]settingsdomain.Setting{
		{Path: []string{"app", "appearance", "mode"}, JSON: `"dark"`},
	})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), `"id": "s0a1b2c3d"`) && !strings.Contains(string(after), `"id":"s0a1b2c3d"`) {
		t.Errorf("the secrets section is gone from:\n%s", after)
	}
}

func TestAWriteAtTheSecretsSectionIsRejectedAndTheFileIsUnchanged(t *testing.T) {
	path := write(t, `{`+validSecretsSection+`}`)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := container.NewConfig(container.Config{
		Paths: container.Paths{SettingsPath: path},
	})

	for _, target := range [][]string{{"secrets"}, {"secrets", "0", "allowed"}} {
		err := container.GetSettingsWriter(cfg)([]settingsdomain.Setting{{Path: target, JSON: `[]`}})
		if err == nil {
			t.Errorf("a write at %v was accepted", target)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Errorf("the file changed:\n%s", after)
	}
}
