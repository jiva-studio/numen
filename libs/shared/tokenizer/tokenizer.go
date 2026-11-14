// Package tokenizer turns text into the token ids of a model, as Hugging Face `tokenizers` does for
// its tokenizer.json.
package tokenizer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

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
	// GetVersion returns Version for a tokenizer this package encodes, and "" for a delegated one.
	// A caller that stores ids keys them by it, so a change to the encoding invalidates them.
	GetVersion() string
}

// Version names the encoding of the Unigram tokenizer of this package. It changes whenever the
// ids it gives for some text change.
const Version = "unigram-1"

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

// Load reads the tokenizer.json at path. A Unigram model is encoded by this package, and a file
// with a step it does not support fails here, with an error that names the step. Every other model
// type is encoded by hftokenizer of go-huggingface.
func Load(path string) (Tokenizer, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read tokenizer: %w", err)
	}
	var head struct {
		Model struct {
			Type string `json:"type"`
		} `json:"model"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, fmt.Errorf("tokenizer %s: %w", path, err)
	}
	var tok Tokenizer
	if head.Model.Type == "Unigram" {
		tok, err = loadUnigramTokenizer(raw)
	} else {
		tok, err = loadDelegate(raw)
	}
	if err != nil {
		return nil, fmt.Errorf("tokenizer %s: %w", path, err)
	}
	return tok, nil
}

type unigramTokenizer struct {
	normalizers  []normalizer
	preTokenizer preTokenizer
	vocab        *vocabulary
	template     *template
	padID        int
	hasPad       bool
}

func loadUnigramTokenizer(raw []byte) (*unigramTokenizer, error) {
	file, err := parseTokenizerFile(raw)
	if err != nil {
		return nil, err
	}
	vocab, err := newVocabulary(file.Model)
	if err != nil {
		return nil, fmt.Errorf("model: %w", err)
	}
	tok := &unigramTokenizer{vocab: vocab}
	if file.Normalizer != nil {
		if tok.normalizers, err = buildNormalizers(*file.Normalizer); err != nil {
			return nil, fmt.Errorf("normalizer: %w", err)
		}
	}
	if file.PreTokenizer != nil {
		if tok.preTokenizer, err = buildPreTokenizer(*file.PreTokenizer); err != nil {
			return nil, fmt.Errorf("pre_tokenizer: %w", err)
		}
	}
	if file.PostProcessor != nil {
		if tok.template, err = newTemplate(*file.PostProcessor); err != nil {
			return nil, fmt.Errorf("post_processor: %w", err)
		}
	}
	tok.padID, tok.hasPad = findPadID(file.AddedTokens)
	return tok, nil
}

func findPadID(tokens []addedToken) (int, bool) {
	id, found := 0, false
	for _, token := range tokens {
		if token.Content == "<pad>" || token.Content == "[PAD]" {
			id, found = token.ID, true
		}
	}
	return id, found
}

func (u *unigramTokenizer) Encode(text string) []int {
	ids := u.encodePieces(text)
	if u.template == nil {
		return ids
	}
	return u.template.applySingle(ids)
}

func (u *unigramTokenizer) EncodePair(first, second string, maxTokens int) (Encoding, error) {
	if u.template == nil || !u.template.hasPair() {
		return Encoding{}, ErrNoPairTemplate
	}
	budget := maxTokens - u.template.countPairSpecials()
	if budget < 0 {
		return Encoding{}, ErrBudgetTooSmall
	}
	a, b := u.encodePieces(first), u.encodePieces(second)
	keepA, keepB := truncateLongestFirst(len(a), len(b), budget)
	return u.template.applyPair(a[:keepA], b[:keepB]), nil
}

func (u *unigramTokenizer) GetPadID() (int, bool) {
	return u.padID, u.hasPad
}

func (u *unigramTokenizer) GetVersion() string {
	return Version
}

func (u *unigramTokenizer) encodePieces(text string) []int {
	for _, normalize := range u.normalizers {
		text = normalize(text)
	}
	pieces := []string{text}
	if u.preTokenizer != nil {
		pieces = u.preTokenizer(pieces)
	}
	var ids []int
	for _, piece := range pieces {
		ids = u.vocab.appendIDs(ids, piece)
	}
	return ids
}
