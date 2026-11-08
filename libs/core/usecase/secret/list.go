package secret

import (
	"slices"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// Listing is the secrets as the window shows them: the sections of the page in order, the menu of
// every declaration, and where the values are kept when it is a file. No value and no flag about
// a value is part of it, and the key store is not read.
type Listing struct {
	Sections        []Section
	Menus           []Menu
	CredentialsPath string
}

// List reads the secrets from the list, without their values.
type List struct {
	operation
	credentialsPath func() string
}

// Execute returns the sections of the page and the menu of each declaration of the installed plugins.
func (u *List) Execute() (Listing, error) {
	list, err := u.ledger.GetSecrets()
	if err != nil {
		return Listing{}, err
	}
	declarations := u.plugins.ListDeclarations()
	listing := Listing{Sections: ListSections(list, ListAskedGroups(declarations))}
	var seen []pluginport.DeclarationKey
	for _, one := range declarations {
		if slices.Contains(seen, one) {
			continue
		}
		seen = append(seen, one)
		listing.Menus = append(listing.Menus, Menu{Declaration: one, Secrets: ListMenu(listing.Sections, one)})
	}
	if u.credentialsPath != nil {
		listing.CredentialsPath = u.credentialsPath()
	}
	return listing, nil
}
