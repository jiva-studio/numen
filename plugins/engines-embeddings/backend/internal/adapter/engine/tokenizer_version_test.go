package engine

import (
	"testing"

	"github.com/jiva-studio/numen/libs/shared/tokenizer"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

func TestTheDomainNamesTheVersionOfTheSharedUnigramTokenizer(t *testing.T) {
	if model.UnigramTokenizerVersion != tokenizer.Version {
		t.Errorf("model.UnigramTokenizerVersion = %q, tokenizer.Version = %q", model.UnigramTokenizerVersion, tokenizer.Version)
	}
}
