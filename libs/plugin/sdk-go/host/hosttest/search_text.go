package hosttest

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	notesdomain "github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// SearchText answers from the results SetIndexedResults set, in their order. Words and hybrid mode keep
// the results whose text contains every word of the query; a raw words query counts its bare words, with
// the FTS5 operators left out, which is a limit of the fake. Meaning mode keeps the results the embedder
// gives a vector. The Text of each result is cut to MaxExcerptBytes around the first word of the query,
// as the host cuts it. Raw words outside words mode return an error, as the host refuses them.
func (s *Search) SearchText(_ context.Context, query host.SearchResultQuery) ([]host.SearchResult, error) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	one, err := s.openVaultLocked(host.PermissionIndexRead)
	if err != nil {
		return nil, err
	}
	if err := checkSearchQuery(query, s.all.model != nil); err != nil {
		return nil, err
	}
	words := getQueryWords(query.Query, query.IsRawWords)
	if len(words) == 0 {
		return nil, nil
	}
	limit, perSourceLimit := getSearchLimits(query)
	var found []host.SearchResult
	perSource := map[string]int{}
	for _, result := range one.indexed {
		if !notesdomain.HasPathPrefix(result.Path, query.PathPrefix) || !hasKind(result.Path, query.Kinds) ||
			!s.isFoundLocked(result, words, query.Mode) || perSource[result.Path] >= perSourceLimit {
			continue
		}
		perSource[result.Path]++
		result.Text = getExcerpt(result.Text, findHit(result.Text, words))
		found = append(found, result)
		if len(found) == limit {
			break
		}
	}
	return found, nil
}

func checkSearchQuery(query host.SearchResultQuery, hasModel bool) error {
	if err := checkKinds(query.Kinds); err != nil {
		return err
	}
	if query.IsRawWords && query.Mode != host.SearchModeWords {
		return errors.New("host: raw words need SearchModeWords")
	}
	if query.Mode == host.SearchModeMeaning && !hasModel {
		return host.ErrNoModel
	}
	return nil
}

func getSearchLimits(query host.SearchResultQuery) (limit, perSourceLimit int) {
	limit = host.DefaultSearchedResults
	if query.Limit > 0 {
		limit = min(query.Limit, host.MaxSearchedResults)
	}
	perSourceLimit = host.DefaultSearchedResultsPerSource
	if query.ResultsPerSource > 0 {
		perSourceLimit = query.ResultsPerSource
	}
	return limit, perSourceLimit
}

func (s *Search) isFoundLocked(result host.SearchResult, words []string, mode host.SearchMode) bool {
	if mode == host.SearchModeMeaning {
		return s.all.embed != nil && s.all.embed(result.Text) != nil
	}
	return hasEveryWord(result.Text, words)
}

func getQueryWords(query string, isRaw bool) []string {
	words := getWords(query)
	if !isRaw {
		return words
	}
	return slices.DeleteFunc(words, func(word string) bool {
		_, err := strconv.Atoi(word)
		return err == nil || word == "and" || word == "or" || word == "not" || word == "near"
	})
}

func hasKind(path string, kinds []domain.SourceKind) bool {
	if len(kinds) == 0 {
		return true
	}
	kind := domain.KindFile
	if strings.HasSuffix(path, domain.NoteExtension) {
		kind = domain.KindNote
	}
	return slices.Contains(kinds, kind)
}

func hasEveryWord(text string, words []string) bool {
	in := getWords(text)
	return !slices.ContainsFunc(words, func(word string) bool { return !slices.Contains(in, word) })
}

func getWords(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// SetIndexedResults sets the results in the index of the vault, in the order it lists them. A
// SearchText call returns those whose text contains every word of the query.
func (s *Search) SetIndexedResults(results []host.SearchResult) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	if one, ok := s.lookupVaultLocked(); ok {
		one.indexed = slices.Clone(results)
	}
}

func checkKinds(kinds []domain.SourceKind) error {
	for _, kind := range kinds {
		if kind != domain.KindNote && kind != domain.KindFile {
			return fmt.Errorf("invalid source kind %q: use %q or %q", kind, domain.KindNote, domain.KindFile)
		}
	}
	return nil
}

func findHit(text string, words []string) int {
	lower := strings.ToLower(text)
	best := -1
	for _, word := range words {
		if pos := strings.Index(lower, word); pos >= 0 && (best < 0 || pos < best) {
			best = pos
		}
	}
	return max(best, 0)
}

func getExcerpt(text string, hit int) string {
	if len(text) <= host.MaxExcerptBytes {
		return text
	}
	start := min(max(hit-host.MaxExcerptBytes/4, 0), len(text))
	end := min(start+host.MaxExcerptBytes, len(text))
	for start < end && !utf8.RuneStart(text[start]) {
		start++
	}
	for end < len(text) && end > start && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[start:end]
}
