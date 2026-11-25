package searchquery

import (
	"context"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// SearchPreparer creates the use case over the index of one vault.
type SearchPreparer = func(ctx context.Context, v domain.Vault) (*FindSearchResults, error)

// FindSearchResultsByName runs a query within the vault a person names.
type FindSearchResultsByName struct {
	lookup  vaultport.VaultLookup
	prepare SearchPreparer
}

// NewFindSearchResultsByName creates the use case that resolves the vault with find and runs the query
// with the use case prepare returns.
func NewFindSearchResultsByName(lookup vaultport.VaultLookup, prepare SearchPreparer) FindSearchResultsByName {
	return FindSearchResultsByName{lookup: lookup, prepare: prepare}
}

// ExecuteByName resolves the vault, prepares the search over it and returns the results of the query.
func (u FindSearchResultsByName) ExecuteByName(ctx context.Context, nameOrPath string, r searchquery.Request) (searchquery.SearchResults, error) {
	v, err := vaultdomain.FindByName(nameOrPath, u.lookup.FindVault)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	find, err := u.prepare(ctx, v)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	return find.Execute(ctx, v, r)
}
