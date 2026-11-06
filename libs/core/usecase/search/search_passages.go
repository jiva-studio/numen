package search

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

const (
	// MaxSearchedPassages is how many passages one SearchPassages call returns at most, and
	// MaxPassageBytes how many bytes of text each of them carries: a window around the hit.
	MaxSearchedPassages = 100
	MaxPassageBytes     = 1200
	// passagesEach is how many passages of one source a SearchPassages call returns at most.
	passagesEach = 3
)

// PassageFinder is the search of the passages of a vault, which Search is.
type PassageFinder interface {
	Execute(ctx context.Context, v domain.Vault, query string, p Parameters) ([]domain.Passage, error)
}

// PassageQuery is a question to the passages of a vault.
type PassageQuery struct {
	Query string
	// Limit is how many passages to return. Zero or less takes the default of the search, and a
	// limit above MaxSearchedPassages is MaxSearchedPassages.
	Limit int
	// Kinds are the kinds of source to look in. Empty looks in every one.
	Kinds []domain.SourceKind
	// PathPrefix limits the sources to those whose path starts with it. Empty limits none.
	PathPrefix string
}

// SearchPassages finds the passages of a vault that answer a query, by the words and by the meaning,
// with the parameters of the window search.
type SearchPassages struct {
	finder PassageFinder
}

// NewSearchPassages returns the passage search over the finder.
func NewSearchPassages(finder PassageFinder) *SearchPassages {
	return &SearchPassages{finder: finder}
}

// Execute returns the passages that answer the query, best first, at most passagesEach of one source.
// An installation with no model answers by the words alone, and an empty query finds nothing. The
// text of each passage is cut to the window around the hit, and its range is the whole passage.
func (u *SearchPassages) Execute(ctx context.Context, v domain.Vault, q PassageQuery) ([]domain.Passage, error) {
	query := strings.TrimSpace(q.Query)
	if query == "" {
		return nil, nil
	}
	parameters := GetTypingParameters(Hybrid, min(max(q.Limit, 0), MaxSearchedPassages))
	parameters.IsGrowing = false
	parameters.Each = passagesEach
	parameters.Kinds = q.Kinds
	parameters.PathPrefix = q.PathPrefix
	found, err := u.finder.Execute(ctx, v, query, parameters)
	if err != nil {
		return nil, err
	}
	for i := range found {
		found[i].Text = getWindow(found[i].Text, found[i].HitAt)
	}
	return found, nil
}

// getWindow is at most MaxPassageBytes of the text around the hit, from a quarter of that before it,
// and never splits a character.
func getWindow(text string, hitAt int) string {
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
