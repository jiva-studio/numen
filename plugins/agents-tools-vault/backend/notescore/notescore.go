// Package notescore ranks notes against the words of a search.
package notescore

import (
	"strings"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// Query is the text of a search, lower-cased, and its words.
type Query struct {
	phrase string
	words  []string
}

// ParseQuery reads the text of a search; the text is trimmed and lower-cased.
func ParseQuery(text string) Query {
	phrase := strings.ToLower(strings.TrimSpace(text))
	return Query{phrase: phrase, words: strings.Fields(phrase)}
}

// IsEmpty reports whether the query has no characters.
func (q Query) IsEmpty() bool {
	return q.phrase == ""
}

// GetLimit returns the number of matches to return: the limit, 20 when it is not positive, at most 100.
func GetLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultLimit
	case limit > maxLimit:
		return maxLimit
	}
	return limit
}

// ScoreName scores a note by how its title and path contain the query.
func (q Query) ScoreName(title, path string) int {
	score := 0
	titleLower := strings.ToLower(title)
	pathLower := strings.ToLower(path)
	if strings.Contains(titleLower, q.phrase) {
		score += 100
	}
	if strings.Contains(pathLower, q.phrase) {
		score += 50
	}
	for _, word := range q.words {
		if len(word) > 1 {
			if strings.Contains(titleLower, word) {
				score += 25
			}
			if strings.Contains(pathLower, word) {
				score += 15
			}
		}
	}
	return score
}

// ScoreText scores the whole text of a note, and finds in its body the first line that holds the query,
// else the first line that holds a word of it. The line counts from 1; it is 0 and the snippet empty
// when no line holds any.
func (q Query) ScoreText(text, body string) (score int, snippet string, line int) {
	textLower := strings.ToLower(text)
	if strings.Contains(textLower, q.phrase) {
		score += 40
	}
	for _, word := range q.words {
		if len(word) > 1 && strings.Contains(textLower, word) {
			score += 10
		}
	}
	snippet, line = q.findSnippet(body)
	return score, snippet, line
}

func (q Query) findSnippet(body string) (snippet string, line int) {
	for index, text := range strings.Split(body, "\n") {
		lower := strings.ToLower(text)
		if strings.Contains(lower, q.phrase) {
			return strings.TrimSpace(text), index + 1
		}
		if snippet == "" && q.hasWord(lower) {
			snippet, line = strings.TrimSpace(text), index+1
		}
	}
	return snippet, line
}

func (q Query) hasWord(lower string) bool {
	for _, word := range q.words {
		if len(word) > 1 && strings.Contains(lower, word) {
			return true
		}
	}
	return false
}
