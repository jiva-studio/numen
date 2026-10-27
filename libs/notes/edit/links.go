package edit

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"
)

// PointAt makes a note name one place under a link `type`, in one read and one write. An empty
// address takes the entry out, so the note names none.
//
// The entry is written where a feature reads it: one type, one place. What the person wrote on it —
// its role, its label, why it exists — is kept, and every other entry of the block comes out of the
// write as the bytes it went in as.
//
// The edit's file stat, when it is given, is what the caller believes is on disk. A note that has
// changed since it was read is left alone and ErrStale comes back. A write that landed answers with
// the file stat of the file it produced.
func (e Edit) PointAt(
	ctx context.Context, v domain.Vault, from, of string, to domain.Address, role domain.LinkRole,
) (domain.FileStat, error) {
	if of == "" {
		return domain.FileStat{}, errors.New("a link is pointed at under a type")
	}
	if to.Value != "" && !domain.IsKnownRole(role) {
		return domain.FileStat{}, fmt.Errorf("%q is not a role a link can carry", role)
	}
	return e.Apply(ctx, v, from, func(doc *markdown.Document) error {
		return doc.SetLinkOfType(of, to, role)
	})
}
