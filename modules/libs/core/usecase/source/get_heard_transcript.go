package source

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
)

// GetHeardTranscript reads the text a model wrote from the file at a path.
type GetHeardTranscript struct {
	texts sourceTexts
}

func (u GetHeardTranscript) Execute(ctx context.Context, v domain.Vault, path string) (Transcript, error) {
	stored, err := u.texts.getStoredText(ctx, v, path)
	if err != nil || !stored.isStored {
		return Transcript{}, err
	}
	producer, hash := stored.meta.Producer, stored.meta.Hash
	words, err := readFirst(ctx, stored.store, text.Artifact(producer, hash), text.Partial(producer, hash))
	return Transcript{Words: words, Exists: true}, err
}
