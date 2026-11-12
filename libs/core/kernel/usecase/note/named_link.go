package note

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// ErrInvalidLink is a link that names no note to go to, or whose target has no address.
var ErrInvalidLink = errors.New("the link does not name a note")

// NamedLink is a link to the note at a path of the vault, written as the name of the note or the address
// that the index says is unique.
type NamedLink struct {
	To     string
	Role   domain.LinkRole
	Type   string
	Why    string
	Label  string
	Marker domain.LinkMarker
}

func resolveNamedLink(ctx context.Context, names edit.NameQueries, v domain.Vault, link NamedLink) (domain.Link, error) {
	if link.To == "" {
		return domain.Link{}, fmt.Errorf("%w: a link needs a note to go to", ErrInvalidLink)
	}
	target, err := getLinkAddress(ctx, names, v, link.To)
	if err != nil {
		return domain.Link{}, fmt.Errorf("%w: %w", ErrInvalidLink, err)
	}
	return domain.Link{
		Target: target, Role: link.Role, Type: link.Type, Why: link.Why, Label: link.Label, Marker: link.Marker,
	}, nil
}

func resolveNamedLinks(ctx context.Context, names edit.NameQueries, v domain.Vault, links []NamedLink) ([]domain.Link, error) {
	out := make([]domain.Link, 0, len(links))
	for _, one := range links {
		link, err := resolveNamedLink(ctx, names, v, one)
		if err != nil {
			return nil, err
		}
		out = append(out, link)
	}
	return out, nil
}
