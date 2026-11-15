package searchquery

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// VaultFinding resolves a vault from the name or the path a person gives.
type VaultFinding = func(nameOrPath string) (domain.Vault, error)

// SearchPreparing creates the use case over the index of one vault.
type SearchPreparing = func(ctx context.Context, v domain.Vault) (*FindSearchResults, error)

// FindSearchResultsByName runs a query within the vault a person names.
type FindSearchResultsByName struct {
	find    VaultFinding
	prepare SearchPreparing
}

// NewFindSearchResultsByName creates the use case that resolves the vault with find and runs the query
// with the use case prepare returns.
func NewFindSearchResultsByName(find VaultFinding, prepare SearchPreparing) FindSearchResultsByName {
	return FindSearchResultsByName{find: find, prepare: prepare}
}

// ExecuteByName resolves the vault, prepares the search over it and returns the results of the query.
func (u FindSearchResultsByName) ExecuteByName(ctx context.Context, nameOrPath string, r Request) (searchquery.SearchResults, error) {
	v, err := u.find(nameOrPath)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	find, err := u.prepare(ctx, v)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	return find.Execute(ctx, v, r)
}
