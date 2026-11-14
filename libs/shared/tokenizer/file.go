package tokenizer

import (
	"encoding/json"
	"slices"
)

// tokenizerFile is the part of tokenizer.json a Unigram tokenizer reads. The truncation and padding
// objects, the decoder and the offsets are left to the caller.
type tokenizerFile struct {
	AddedTokens   []addedToken `json:"added_tokens"`
	Normalizer    *step        `json:"normalizer"`
	PreTokenizer  *step        `json:"pre_tokenizer"`
	PostProcessor *step        `json:"post_processor"`
	Model         unigramModel `json:"model"`
}

type addedToken struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

type unigramModel struct {
	UnkID        *int                `json:"unk_id"`
	ByteFallback bool                `json:"byte_fallback"`
	Vocab        [][]json.RawMessage `json:"vocab"`
}

// step is one normalizer, pre-tokenizer or post-processor. Its settings are decoded once its type
// is known, so a step of an unsupported type is refused by name rather than by the shape of its fields.
type step struct {
	Type string
	raw  json.RawMessage
}

func (s *step) UnmarshalJSON(data []byte) error {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return err
	}
	s.Type, s.raw = head.Type, slices.Clone(data)
	return nil
}

func (s step) decode(settings any) error {
	return json.Unmarshal(s.raw, settings)
}
