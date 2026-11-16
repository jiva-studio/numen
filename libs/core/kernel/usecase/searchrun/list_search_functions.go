package searchrun

import "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"

// ListSearchFunctions returns the catalog of the query language and its revision.
type ListSearchFunctions struct {
	catalog func() searchquery.Catalog
}

func (u *ListSearchFunctions) Execute() (searchquery.Catalog, string) {
	catalog := u.catalog()
	return catalog, searchquery.GetCatalogRevision(catalog)
}
