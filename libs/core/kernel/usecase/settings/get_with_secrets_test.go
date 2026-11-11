package settings_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func newGet() *settings.Get {
	return settings.NewGet(func() (string, string, error) { return `{"v":1}`, "/home/numen.json", nil }, nil, nil, nil)
}

func TestTheSettingsPageIncludesTheSecrets(t *testing.T) {
	lister := func(context.Context) ([]plugin.SecretSection, []plugin.SecretMenu, string, error) {
		return nil, nil, "/home/credentials.json", nil
	}
	got, err := settings.NewGetWithSecrets(newGet(), lister).Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "/home/numen.json" || got.Written != `{"v":1}` || got.Secrets.CredentialsPath != "/home/credentials.json" {
		t.Errorf("read %+v", got)
	}
}

func TestABuildWithNoSecretsAnswersTheSettingsAlone(t *testing.T) {
	got, err := settings.NewGetWithSecrets(newGet(), nil).Execute(t.Context())
	if err != nil || got.Path != "/home/numen.json" || len(got.Secrets.Sections) != 0 {
		t.Errorf("read %+v, %v", got, err)
	}
}

func TestASecretListThatFailsFailsThePage(t *testing.T) {
	broken := errors.New("key store locked")
	lister := func(context.Context) ([]plugin.SecretSection, []plugin.SecretMenu, string, error) {
		return nil, nil, "", broken
	}
	if _, err := settings.NewGetWithSecrets(newGet(), lister).Execute(t.Context()); !errors.Is(err, broken) {
		t.Errorf("answered %v", err)
	}
}
