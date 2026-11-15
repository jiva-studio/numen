package engine

import (
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/tokenizer"
)

// pairTokenizer encodes a (query, document) pair as the pair template of the tokenizer of a model says,
// cut to the tokens of the profile.
type pairTokenizer struct {
	tokens    tokenizer.Tokenizer
	maxTokens int
	pad       int
}

func loadPairTokenizer(path string, maxTokens int) (*pairTokenizer, error) {
	tokens, err := tokenizer.Load(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if _, err := tokens.EncodePair("", "", maxTokens); err != nil {
		if errors.Is(err, tokenizer.ErrNoPairTemplate) {
			return nil, fmt.Errorf("%s: tokenizer.json has no pair template: %w", path, err)
		}
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	pad, ok := tokens.GetPadID()
	if !ok {
		pad = 0
	}
	return &pairTokenizer{tokens: tokens, maxTokens: maxTokens, pad: pad}, nil
}

func (t *pairTokenizer) encode(query, document string) (tokenizer.Encoding, error) {
	return t.tokens.EncodePair(query, document, t.maxTokens)
}
