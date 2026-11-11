package settings

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// SecretLister lists the secrets without their values.
type SecretLister interface {
	Execute(ctx context.Context) (plugin.SecretListing, error)
}

// NoSecrets lists nothing, for a build that serves no secrets.
type NoSecrets struct{}

// Execute lists no secret.
func (NoSecrets) Execute(context.Context) (plugin.SecretListing, error) {
	return plugin.SecretListing{}, nil
}

// CurrentWithSecrets is the settings of the installation and the secrets of its plugins.
type CurrentWithSecrets struct {
	Current
	Secrets plugin.SecretListing
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

// Execute reads the settings, then the secrets.
func (u *GetWithSecrets) Execute(ctx context.Context) (CurrentWithSecrets, error) {
	current, err := u.get.Execute()
	if err != nil {
		return CurrentWithSecrets{}, err
	}
	listing, err := u.secrets.Execute(ctx)
	if err != nil {
		return CurrentWithSecrets{}, err
	}
	return CurrentWithSecrets{Current: current, Secrets: listing}, nil
}
