package edit

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/markdown"
)

// PointAt makes a note name one place under a link `type`, in one read and one write.
// An empty address takes the entry out. The entry keeps the user's role, label and
// reason, and every other entry is written back byte for byte.
// A note that changed since the edit's file stat returns ErrStale and is left unchanged.
// A successful write returns the file stat it produced.
func (e Edit) PointAt(
	ctx context.Context, v domain.Vault, from, of string, to domain.Address, role domain.LinkRole,
) (domain.FileStat, error) {
	if of == "" {
		return domain.FileStat{}, errors.New("a link is pointed at under a type")
	}
	if to.Value != "" && !domain.IsKnownRole(role) {
		return domain.FileStat{}, fmt.Errorf("%q is not a role a link can have", role)
	}
	return e.Apply(ctx, v, from, func(doc *markdown.Document) error {
		return doc.SetLinkOfType(of, to, role)
	})
}
