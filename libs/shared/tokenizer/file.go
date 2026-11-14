package tokenizer

import (
	"encoding/json"
	"fmt"
)

// tokenizerFile is the part of tokenizer.json a Unigram tokenizer reads. The truncation and padding
// objects, the decoder and the offsets are left to the caller. tokenizer.json is the format of
// Hugging Face, whose keys are snake_case, so it is read by key through jsonObject.
type tokenizerFile struct {
	AddedTokens   []addedToken
	Normalizer    *step
	PreTokenizer  *step
	PostProcessor *step
	Model         unigramModel
}

type addedToken struct {
	ID      int
	Content string
}

type unigramModel struct {
	UnkID        *int
	ByteFallback bool
	Vocab        [][]json.RawMessage
}

// jsonObject is a JSON object read key by key.
type jsonObject map[string]json.RawMessage

func (o jsonObject) read(key string, out any) error {
	raw, ok := o[key]
	if !ok {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	return nil
}

func parseTokenizerFile(raw []byte) (tokenizerFile, error) {
	var file tokenizerFile
	var root, model jsonObject
	if err := json.Unmarshal(raw, &root); err != nil {
		return file, err
	}
	for _, field := range []struct {
		key string
		out any
	}{
		{"added_tokens", &file.AddedTokens},
		{"normalizer", &file.Normalizer},
		{"pre_tokenizer", &file.PreTokenizer},
		{"post_processor", &file.PostProcessor},
		{"model", &model},
	} {
		if err := root.read(field.key, field.out); err != nil {
			return file, err
		}
	}
	for _, field := range []struct {
		key string
		out any
	}{
		{"unk_id", &file.Model.UnkID},
		{"byte_fallback", &file.Model.ByteFallback},
		{"vocab", &file.Model.Vocab},
	} {
		if err := model.read(field.key, field.out); err != nil {
			return file, fmt.Errorf("model: %w", err)
		}
	}
	return file, nil
}

// step is one normalizer, pre-tokenizer or post-processor. Its settings are read once its type is
// known, so a step of an unsupported type is refused by name rather than by the shape of its fields.
type step struct {
	Type   string
	fields jsonObject
}

func (s *step) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &s.fields); err != nil {
		return err
	}
	return s.fields.read("type", &s.Type)
}
