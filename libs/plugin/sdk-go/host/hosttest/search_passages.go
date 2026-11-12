package hosttest

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"

	notesdomain "github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

func (s *Search) SearchPassages(_ context.Context, query host.PassageQuery) ([]host.Passage, error) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	one, err := s.openVaultLocked(host.PermissionIndexRead)
	if err != nil {
		return nil, err
	}
	if err := checkKinds(query.Kinds); err != nil {
		return nil, err
	}
	words := getWords(query.Query)
	if len(words) == 0 {
		return nil, nil
	}
	limit := host.DefaultSearchedPassages
	if query.Limit > 0 {
		limit = min(query.Limit, host.MaxSearchedPassages)
	}
	var found []host.Passage
	perSource := map[string]int{}
	for _, passage := range one.indexed {
		if !notesdomain.HasPathPrefix(passage.Path, query.PathPrefix) || !hasKind(passage.Path, query.Kinds) ||
			!hasEveryWord(passage.Text, words) || perSource[passage.Path] >= host.MaxSearchedPassagesPerSource {
			continue
		}
		perSource[passage.Path]++
		found = append(found, passage)
		if len(found) == limit {
			break
		}
	}
	return found, nil
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

// SetIndexedPassages sets the passages in the index of the vault, in the order it lists them. A
// SearchPassages call returns those whose text contains every word of the query.
func (s *Search) SetIndexedPassages(passages []host.Passage) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	if one, ok := s.lookupVaultLocked(); ok {
		one.indexed = slices.Clone(passages)
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
