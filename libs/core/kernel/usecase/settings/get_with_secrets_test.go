package settings_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

type secrets struct {
	listing plugin.SecretListing
	err     error
}

func (s secrets) Execute(context.Context) (plugin.SecretListing, error) { return s.listing, s.err }

func newGet() *settings.Get {
	return settings.NewGet(func() (string, string, error) { return `{"v":1}`, "/home/numen.json", nil }, nil, nil, nil)
}

func TestTheSettingsPageIncludesTheSecrets(t *testing.T) {
	listing := plugin.SecretListing{CredentialsPath: "/home/credentials.json"}
	got, err := settings.NewGetWithSecrets(newGet(), secrets{listing: listing}).Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "/home/numen.json" || got.Written != `{"v":1}` || got.Secrets.CredentialsPath != "/home/credentials.json" {
		t.Errorf("read %+v", got)
	}
}

func TestABuildWithNoSecretsAnswersTheSettingsAlone(t *testing.T) {
	got, err := settings.NewGetWithSecrets(newGet(), settings.NoSecrets{}).Execute(t.Context())
	if err != nil || got.Path != "/home/numen.json" || len(got.Secrets.Sections) != 0 {
		t.Errorf("read %+v, %v", got, err)
	}
}

func TestASecretListThatFailsFailsThePage(t *testing.T) {
	broken := errors.New("key store locked")
	if _, err := settings.NewGetWithSecrets(newGet(), secrets{err: broken}).Execute(t.Context()); !errors.Is(err, broken) {
		t.Errorf("answered %v", err)
	}
}
