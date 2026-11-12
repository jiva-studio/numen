package settings

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// SecretListing is the secrets as the window shows them: the sections of the page in order, the menu of
// every declaration, and where the values are kept when it is a file. No value and no flag about a
// value is part of it, and the key store is not read.
type SecretListing struct {
	Sections        []plugin.SecretSection
	Menus           []plugin.SecretMenu
	CredentialsPath string
}

// SecretLister lists the secrets without their values.
type SecretLister func(ctx context.Context) ([]plugin.SecretSection, []plugin.SecretMenu, string, error)

// CurrentWithSecrets is the settings of the installation and the secrets of its plugins.
type CurrentWithSecrets struct {
	Current
	Secrets SecretListing
}

// GetWithSecrets reads every setting with the secrets the plugins declare, as the settings page shows them.
type GetWithSecrets struct {
	get     *Get
	secrets SecretLister
}

// NewGetWithSecrets creates the reading over the settings and the list of secrets.
func NewGetWithSecrets(get *Get, secrets SecretLister) *GetWithSecrets {
	return &GetWithSecrets{get: get, secrets: secrets}
}

func (u *GetWithSecrets) Execute(ctx context.Context) (CurrentWithSecrets, error) {
	current, err := u.get.Execute()
	if err != nil {
		return CurrentWithSecrets{}, err
	}
	if u.secrets == nil {
		return CurrentWithSecrets{Current: current}, nil
	}
	sections, menus, credentialsPath, err := u.secrets(ctx)
	if err != nil {
		return CurrentWithSecrets{}, err
	}
	return CurrentWithSecrets{
		Current: current,
		Secrets: SecretListing{Sections: sections, Menus: menus, CredentialsPath: credentialsPath},
	}, nil
}
