package cards

import (
	"context"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

// Write puts a deck or a stencil back where it came from.
//
// A deck and a stencil are notes, so this is a note's write with a deck's bound
// on it: the body is replaced whole, the frontmatter is the person's and is not
// rewritten, and a file that has changed since the caller read it is left alone
// with port.ErrChanged.
type Write struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	Index   func(ctx context.Context, v domain.Vault, paths []string) error
	// Now is when this is happening. An identifier written here carries it.
	Now func() time.Time
}

// Deck puts body in the deck at path, and answers with the fingerprint the
// write produced, which is what the caller presents at its next write.
func (u Write) Deck(
	ctx context.Context, v domain.Vault, path, body string, fingerprint domain.FileRef,
) (domain.FileRef, error) {
	if len(body) > MaxBytes {
		return domain.FileRef{}, fmt.Errorf(
			"%w: %d bytes, and %d is the most a deck is", note.ErrTooLarge, len(body), MaxBytes)
	}
	return u.note().Execute(ctx, v, path, body, fingerprint)
}

// Stencil puts body in the stencil at path. A stencil is a note and is bounded
// as one.
func (u Write) Stencil(
	ctx context.Context, v domain.Vault, path, body string, fingerprint domain.FileRef,
) (domain.FileRef, error) {
	if len(body) > note.MaxBytes {
		return domain.FileRef{}, fmt.Errorf(
			"%w: %d bytes, and %d is the most a stencil is", note.ErrTooLarge, len(body), note.MaxBytes)
	}
	return u.note().Execute(ctx, v, path, body, fingerprint)
}

func (u Write) note() note.Write {
	return note.Write{Readers: u.Readers, Writers: u.Writers, Index: u.Index, Now: u.Now}
}
