package source

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// CutSource cuts one source again from whatever its text now says.
//
// A recognition, a transcription or a proofreading writes a batch and asks for
// this, so a source being made answers about what has been written so far.
type CutSource struct {
	books Extract
}

func NewCutSource(books Extract) CutSource {
	return CutSource{books: books}
}

func (u CutSource) Execute(ctx context.Context, v domain.Vault, path string) error {
	if _, err := u.books.ExtractOne(ctx, v, path); err != nil {
		return fmt.Errorf("cutting %s: %w", path, err)
	}
	return nil
}
