package model

import "fmt"

// The kinds of tokenizer a profile names.
const (
	TokenizerUnigram = "unigram"
	TokenizerBPE     = "bpe"
)

// UnigramTokenizerVersion is Version of libs/shared/tokenizer, which the domain does not import. An
// open fails when the two differ.
const UnigramTokenizerVersion = "unigram-1"

// GetTokenizerVersion returns the version of the tokenizer the profile names: UnigramTokenizerVersion
// for the Unigram tokenizer and "" for BPE, whose ids did not change.
func (p Profile) GetTokenizerVersion() string {
	if p.Tokenizer == TokenizerUnigram {
		return UnigramTokenizerVersion
	}
	return ""
}

func checkTokenizerKind(p Profile) error {
	if p.Tokenizer != "" && p.Tokenizer != TokenizerUnigram && p.Tokenizer != TokenizerBPE {
		return fmt.Errorf("%s: tokenizer %q is none of %s, %s", p.ID, p.Tokenizer, TokenizerUnigram, TokenizerBPE)
	}
	return nil
}

func checkTokenizerNamed(p Profile) error {
	if p.Tokenizer == "" {
		return fmt.Errorf("%s: a profile that adds a model needs tokenizer, %s or %s", p.ID, TokenizerUnigram, TokenizerBPE)
	}
	return nil
}
