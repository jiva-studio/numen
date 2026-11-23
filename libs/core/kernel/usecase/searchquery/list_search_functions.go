package searchquery

import "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"

// ListSearchFunctions returns the catalog of the query language and its revision.
type ListSearchFunctions struct {
	getCatalog func() searchquery.Catalog
}

func (u *ListSearchFunctions) Execute() (searchquery.Catalog, string) {
	catalog := u.getCatalog()
	return catalog, searchquery.GetCatalogRevision(catalog)
}
