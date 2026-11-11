package search

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/note/domain"
)

const (
	// MaxSearchedPassages is the maximum number of passages one SearchPassages call returns, and
	// MaxPassageBytes the maximum excerpt size of each of them, around the hit.
	MaxSearchedPassages = 100
	MaxPassageBytes     = 1200
	// maxPassagesPerSource is the maximum number of passages of one source a SearchPassages call returns.
	maxPassagesPerSource = 3
)

type PassageFinder interface {
	Execute(ctx context.Context, v domain.Vault, query string, p Parameters) ([]domain.Passage, error)
}

type PassageQuery struct {
	Query      string
	Limit      int
	Kinds      []domain.SourceKind
	PathPrefix string
}

// SearchPassages finds the passages of a vault matching a query, by keywords and by embedding, with
// the parameters of the window search.
type SearchPassages struct {
	finder PassageFinder
}

// NewSearchPassages returns the passage search over the finder.
func NewSearchPassages(finder PassageFinder) *SearchPassages {
	return &SearchPassages{finder: finder}
}

// Execute returns the passages matching the query, best first, at most maxPassagesPerSource of one
// source. An installation with no model searches by keywords only, and an empty query returns
// nothing. The text of each passage is truncated to the excerpt around the hit, and its range is
// the whole passage.
func (u *SearchPassages) Execute(ctx context.Context, v domain.Vault, q PassageQuery) ([]domain.Passage, error) {
	query := strings.TrimSpace(q.Query)
	if query == "" {
		return nil, nil
	}
	parameters := GetTypingParameters(Hybrid, min(max(q.Limit, 0), MaxSearchedPassages))
	parameters.IsGrowing = false
	parameters.Each = maxPassagesPerSource
	parameters.Kinds = q.Kinds
	parameters.PathPrefix = q.PathPrefix
	found, err := u.finder.Execute(ctx, v, query, parameters)
	if err != nil {
		return nil, err
	}
	for i := range found {
		found[i].Text = getExcerpt(found[i].Text, found[i].HitAt)
	}
	return found, nil
}

func getExcerpt(text string, hitAt int) string {
	if len(text) <= MaxPassageBytes {
		return text
	}
	start := min(max(hitAt-MaxPassageBytes/4, 0), len(text))
	end := min(start+MaxPassageBytes, len(text))
	for start < end && !utf8.RuneStart(text[start]) {
		start++
	}
	for end < len(text) && end > start && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[start:end]
}
