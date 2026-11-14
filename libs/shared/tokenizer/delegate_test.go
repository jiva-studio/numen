package tokenizer_test

import (
	"errors"
	"testing"

	"github.com/gomlx/go-huggingface/tokenizers/api"
	"github.com/gomlx/go-huggingface/tokenizers/hftokenizer"
	"github.com/jiva-studio/numen/libs/shared/tokenizer"
)

const bpePath = "testdata/bpe.json"

// A model type other than Unigram is encoded by hftokenizer, built as engines-embeddings builds it.
func TestLoadDelegatesABPEModelToHFTokenizer(t *testing.T) {
	reference, err := hftokenizer.NewFromFile(nil, bpePath)
	if err != nil {
		t.Fatal(err)
	}
	tok := loadTokenizer(t, bpePath)
	for _, text := range []string{"lower lowest", "slow", "newer ones", "", "low\tlow  low"} {
		checkEncode(t, tok, text, reference.Encode(text))
	}
	if len(reference.Encode("lower")) == 0 {
		t.Fatal("the reference encodes nothing: a comparison against nothing")
	}
	refPad, refErr := reference.SpecialTokenID(api.TokPad)
	if pad, ok := tok.GetPadID(); ok != (refErr == nil) || (ok && pad != refPad) {
		t.Errorf("GetPadID() = %d, %v, want %d, %v", pad, ok, refPad, refErr == nil)
	}
}

func TestEncodePairOfADelegatedModelFails(t *testing.T) {
	tok := loadTokenizer(t, bpePath)
	if got, err := tok.EncodePair("low", "lower", 64); !errors.Is(err, tokenizer.ErrNoPairTemplate) {
		t.Errorf("EncodePair = %v, %v, want ErrNoPairTemplate", got, err)
	}
}
