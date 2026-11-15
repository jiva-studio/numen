package tokenizer

import (
	"github.com/gomlx/go-huggingface/tokenizers/api"
	"github.com/gomlx/go-huggingface/tokenizers/hftokenizer"
)

// delegate encodes a model type other than Unigram through hftokenizer, as engines-embeddings
// builds it. hftokenizer has no pair encoding.
type delegate struct {
	tok *hftokenizer.Tokenizer
}

func loadDelegate(raw []byte) (*delegate, error) {
	tok, err := hftokenizer.NewFromContent(nil, raw)
	if err != nil {
		return nil, err
	}
	return &delegate{tok: tok}, nil
}

func (d *delegate) Encode(text string) []int {
	return d.tok.Encode(text)
}

func (d *delegate) EncodePair(string, string, int) (Encoding, error) {
	return Encoding{}, ErrNoPairTemplate
}

func (d *delegate) GetPadID() (int, bool) {
	id, err := d.tok.SpecialTokenID(api.TokPad)
	return id, err == nil
}

func (d *delegate) GetVersion() string {
	return ""
}
