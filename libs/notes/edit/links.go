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
// The entry is written where a feature reads it: one type, one place. The
// user's content of the entry (its role, its label, its reason) is kept, and
// every other entry of the block is written back byte for byte.
//
// The edit's file stat, when given, is the state the caller believes is on disk.
// A note that changed since it was read is left unchanged and ErrStale is
// returned. A successful write returns the file stat of the file it produced.
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
