package secret

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// List reads the secrets from the list, without their values.
type List struct {
	operation
	credentialsPath func(ctx context.Context) string
}

// Execute returns the sections of the page and the menu of each declaration of the installed plugins.
func (u *List) Execute(ctx context.Context) (plugin.SecretListing, error) {
	list, err := u.ledger.GetSecrets()
	if err != nil {
		return plugin.SecretListing{}, err
	}
	declarations := u.plugins.ListDeclarations()
	listing := plugin.SecretListing{Sections: plugin.ListSecretSections(list, plugin.ListAskedSecretGroups(declarations))}
	var seen []plugin.DeclarationKey
	for _, one := range declarations {
		if slices.Contains(seen, one) {
			continue
		}
		seen = append(seen, one)
		listing.Menus = append(listing.Menus, plugin.SecretMenu{Declaration: one, Secrets: plugin.ListSecretMenu(listing.Sections, one)})
	}
	if u.credentialsPath != nil {
		listing.CredentialsPath = u.credentialsPath(ctx)
	}
	return listing, nil
}
