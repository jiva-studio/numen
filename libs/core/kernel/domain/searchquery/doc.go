// Package searchquery holds the search query language of the core: the catalog of its functions, the tree
// of a parsed query and its normal form, the rules of evaluation, fusion, merging and order of results, and
// the limits of one query. A parser turns the text into a RawNode; Normalize turns that into a Query.
package searchquery
