package searchquery

import "context"

// WordMatcher matches an FTS5 query against texts with the tokenizer of the index.
type WordMatcher interface {
	// MatchWords reports for each text whether it matches the expression, in the order of the texts. An
	// expression FTS5 cannot parse returns source.ErrInvalidWords.
	MatchWords(ctx context.Context, expression string, texts []string) ([]bool, error)
}
