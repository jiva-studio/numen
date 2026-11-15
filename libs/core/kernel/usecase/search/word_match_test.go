package search_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/wordmatch"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
)

func getWordMatchNotes() map[string]string {
	return map[string]string{
		"notes/N1.md": "Café au lait in the morning",
		"notes/N2.md": "CAFE NOIR at night",
		"notes/N3.md": "a naïve heron",
		"notes/N4.md": "Чайтанья во Вриндаване",
		"notes/N5.md": "ЧАЙТАНЬЯ И ГОВИНДА",
	}
}

func TestWordMatchingMatchesTheTextsTheIndexMatchesForTheSameQuery(t *testing.T) {
	c := newCorpus(t)
	notes := getWordMatchNotes()
	v := c.addVault(t, notes)
	paths := make([]string, 0, len(notes))
	for path := range notes {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	texts := make([]string, len(paths))
	for i, path := range paths {
		texts[i] = notes[path]
	}

	for _, query := range []string{"cafe", "CAFÉ", "naive", "чайтанья", "ВРИНДАВАНЕ", `"au lait"`, "heron NOT naive", "чайт*"} {
		indexed, err := c.db.ChunkSearch().FindLexical(t.Context(), v.ID, sourcedomain.LexicalQuery{Text: query, Limit: 50, IsRaw: true})
		if err != nil {
			t.Fatalf("the index rejected %q: %v", query, err)
		}
		fromIndex := sources(indexed)
		slices.Sort(fromIndex)

		matched, err := wordmatch.NewMatcher().MatchWords(t.Context(), query, texts)
		if err != nil {
			t.Fatalf("the matcher rejected %q: %v", query, err)
		}
		var fromMatcher []string
		for i, isMatch := range matched {
			if isMatch {
				fromMatcher = append(fromMatcher, paths[i])
			}
		}

		if !slices.Equal(fromIndex, fromMatcher) {
			t.Errorf("for %q the index matched %v and the matcher %v", query, fromIndex, fromMatcher)
		}
		if query == "cafe" && len(fromIndex) != 2 {
			t.Errorf("the index matched %v for cafe, want the notes with café and CAFE", fromIndex)
		}
	}
}
