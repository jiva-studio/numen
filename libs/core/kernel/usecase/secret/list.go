package secret

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Listing is the secrets as the window shows them: the sections of the page in order, the menu of
// every declaration, and where the values are kept when it is a file. No value and no flag about a
// value is part of it, and the key store is not read.
type Listing struct {
	Sections        []plugin.SecretSection
	Menus           []plugin.SecretMenu
	CredentialsPath string
}

// List reads the secrets from the list, without their values.
type List struct {
	operation
	credentialsPath func(ctx context.Context) string
}

// Execute returns the sections of the page and the menu of each declaration of the installed plugins.
func (u *List) Execute(ctx context.Context) (Listing, error) {
	list, err := u.ledger.GetSecrets()
	if err != nil {
		return Listing{}, err
	}
	declarations := u.plugins.ListDeclarations()
	listing := Listing{Sections: plugin.ListSecretSections(list, plugin.ListAskedSecretGroups(declarations))}
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

// ListForSettings returns the sections, menus, and credentials path for the settings page.
func (u *List) ListForSettings(ctx context.Context) ([]plugin.SecretSection, []plugin.SecretMenu, string, error) {
	listing, err := u.Execute(ctx)
	return listing.Sections, listing.Menus, listing.CredentialsPath, err
}
