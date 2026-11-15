package search

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/note/domain"
)

const (
	// MaxSearchedResults is the maximum number of search results one SearchText call returns, and
	// MaxExcerptBytes the maximum excerpt size of each of them, around the hit.
	MaxSearchedResults = 500
	MaxExcerptBytes    = 1200
	// DefaultSearchedResultsPerSource is the number of search results of one source a SearchText call
	// returns when the query names none.
	DefaultSearchedResultsPerSource = 3
)

type ResultFinder interface {
	Execute(ctx context.Context, v domain.Vault, query string, p Parameters) ([]domain.SearchResult, error)
}

// SearchResultQuery is a text search request. Mode selects the participants, Hybrid by default,
// IsRawWords passes Query to the words as an FTS5 query, and IsWholeText keeps the whole text of each
// chunk in WholeText.
type SearchResultQuery struct {
	Query            string
	Limit            int
	ResultsPerSource int
	Kinds            []domain.SourceKind
	PathPrefix       string
	Mode             Mode
	IsRawWords       bool
	IsWholeText      bool
}

// SearchText finds the search results of a vault matching a query, by keywords and by embedding, with
// the parameters of the window search.
type SearchText struct {
	finder ResultFinder
}

// NewSearchText returns the search result search over the finder.
func NewSearchText(finder ResultFinder) *SearchText {
	return &SearchText{finder: finder}
}

// Execute returns the search results matching the query, best first, at most q.Limit of them, capped at
// MaxSearchedResults, and at most q.ResultsPerSource of one source, DefaultSearchedResultsPerSource
// where it is not positive and q.Limit where it is above. An installation with no model searches by
// keywords only, and an empty query returns nothing. The text of each search result is truncated to the
// excerpt around the hit, and its range is the whole search result; with IsWholeText, WholeText holds the
// text before the cut.
func (u *SearchText) Execute(ctx context.Context, v domain.Vault, q SearchResultQuery) ([]domain.SearchResult, error) {
	query := strings.TrimSpace(q.Query)
	if query == "" {
		return nil, nil
	}
	parameters := GetTypingParameters(q.Mode, min(max(q.Limit, 0), MaxSearchedResults))
	parameters.IsGrowing = false
	parameters.IsRawWords = q.IsRawWords
	parameters.Each = getResultsPerSource(q.ResultsPerSource, parameters.Limit)
	parameters.Kinds = q.Kinds
	parameters.PathPrefix = q.PathPrefix
	found, err := u.finder.Execute(ctx, v, query, parameters)
	if err != nil {
		return nil, err
	}
	for i := range found {
		found[i].Text = getExcerpt(found[i].Text, found[i].MatchOffset)
	}
	return found, nil
}

func getResultsPerSource(perSource, limit int) int {
	if perSource <= 0 {
		perSource = DefaultSearchedResultsPerSource
	}
	return min(perSource, limit)
}

func getExcerpt(text string, matchOffset int) string {
	if len(text) <= MaxExcerptBytes {
		return text
	}
	start := min(max(matchOffset-MaxExcerptBytes/4, 0), len(text))
	end := min(start+MaxExcerptBytes, len(text))
	for start < end && !utf8.RuneStart(text[start]) {
		start++
	}
	for end < len(text) && end > start && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[start:end]
}
