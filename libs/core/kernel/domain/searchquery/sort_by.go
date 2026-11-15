package searchquery

import (
	"cmp"
	"slices"
)

// SortKey is what a sortBy stage orders by.
type SortKey int

const (
	SortScore SortKey = iota
	SortPath
	SortTitle
	SortModified
	SortSize
)

// SortStage orders the results by a key of their document.
type SortStage struct {
	Key          SortKey
	IsDescending bool
}

func listSortKeys() []string { return []string{"score", "path", "title", "modified", "size"} }

// ParseSortKey returns the key a sortBy literal names, and false for another literal.
func ParseSortKey(text string) (SortKey, bool) {
	index := slices.Index(listSortKeys(), text)
	return SortKey(index), index >= 0
}

// ParseSortDirection returns true for "desc", false for "asc", and false in the second value for another
// literal.
func ParseSortDirection(text string) (isDescending, isValid bool) {
	switch text {
	case "asc":
		return false, true
	case "desc":
		return true, true
	}
	return false, false
}

// IsDescendingByDefault reports whether the key orders highest first when no direction is named.
func (k SortKey) IsDescendingByDefault() bool {
	return k == SortScore || k == SortModified || k == SortSize
}

// ApplySort orders the results by the key of their document, the score by the result's own; equal keys go
// by rounded score, highest first and none last, then by path, then by start.
func ApplySort(r SearchResults, s SortStage) SearchResults {
	r.Results = slices.Clone(r.Results)
	slices.SortStableFunc(r.Results, func(a, b SearchResult) int {
		order := compareKeys(a, b, s.Key)
		if s.IsDescending {
			order = -order
		}
		return cmp.Or(
			order,
			compareScores(a.Score, b.Score),
			cmp.Compare(a.Document.Path, b.Document.Path),
			cmp.Compare(a.Start, b.Start),
		)
	})
	return r
}

func compareKeys(a, b SearchResult, key SortKey) int {
	switch key {
	case SortScore:
		return -compareScores(a.Score, b.Score)
	case SortPath:
		return cmp.Compare(a.Document.Path, b.Document.Path)
	case SortTitle:
		return cmp.Compare(a.Document.Title, b.Document.Title)
	case SortModified:
		return a.Document.ModifiedAt.Compare(b.Document.ModifiedAt)
	case SortSize:
		return cmp.Compare(a.Document.Size, b.Document.Size)
	}
	return 0
}

// ApplyTop keeps the first n results.
func ApplyTop(r SearchResults, n int) SearchResults {
	if len(r.Results) > n {
		r.Results = r.Results[:n]
	}
	return r
}
