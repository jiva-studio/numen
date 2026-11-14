// Package tokenizer turns text into the token ids of a model, as Hugging Face `tokenizers` does for
// its tokenizer.json.
package tokenizer

import "errors"

// Tokenizer turns text into the token ids of one model, as Hugging Face `tokenizers` does for its
// tokenizer.json. It is safe for concurrent use; nothing changes after Load.
type Tokenizer interface {
	// Encode returns the ids of text inside the single template of the post-processor
	// (<s> A </s> for XLM-R). It does not truncate.
	Encode(text string) []int
	// EncodePair returns the ids of first and second inside the pair template
	// (<s> A </s></s> B </s> for XLM-R), truncated longest_first so that the result, special
	// tokens included, holds at most maxTokens ids.
	EncodePair(first, second string, maxTokens int) (Encoding, error)
	// GetPadID returns the id of the padding token, and false when the file declares none.
	GetPadID() (int, bool)
}

// Encoding is a pair encoded for a cross-encoder.
type Encoding struct {
	IDs     []int
	TypeIDs []int // the type_id of each id, from the template
}

var (
	// ErrNoPairTemplate is returned by EncodePair on a tokenizer whose post-processor has no pair
	// template, and on every delegated tokenizer.
	ErrNoPairTemplate = errors.New("tokenizer has no pair template")
	// ErrBudgetTooSmall is returned by EncodePair when maxTokens is below the number of special
	// tokens of the pair template.
	ErrBudgetTooSmall = errors.New("maxTokens is below the special tokens of the pair template")
)

// Load reads the tokenizer.json at path.
func Load(path string) (Tokenizer, error) {
	return nil, errors.New("tokenizer.Load is not implemented")
}
