package engine

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/tokenizer"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

func loadTokenizer(identity model.ModelInfo, path string) (tokenizer.Tokenizer, error) {
	encoder, err := tokenizer.Load(path)
	if err != nil {
		return nil, err
	}
	if got, want := encoder.GetVersion(), identity.Handling.Tokenizer; got != want {
		return nil, fmt.Errorf("%s: %s gives tokenizer version %q, the tokenizer of the profile gives %q",
			identity.Name, path, got, want)
	}
	return encoder, nil
}

// GetTokenizerVersion returns the version of the tokenizer the model reads texts with, empty when its
// ids are the ones the plugin gave before versions were kept.
func (e *Embedder) GetTokenizerVersion() string {
	return e.encoder.GetVersion()
}
