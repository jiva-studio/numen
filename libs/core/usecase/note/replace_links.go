package note

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"
)

// ErrNotAPair is what a set of links gets when one of them does not run between
// the two notes it was given for.
var ErrNotAPair = errors.New("a link must run between the two notes given")

// PairLink is one relationship between two notes: the note it is written in,
// the note it points to, and its role. The record is addressed by its target and
// Type; the reason already written on it is kept when Why is empty.
type PairLink struct {
	From   string
	To     string
	Role   domain.LinkRole
	Type   string
	Why    string
	Label  string
	Marker domain.LinkMarker
}

// ReplaceLinks makes the links between two notes exactly the ones given.
type ReplaceLinks struct {
	linking   EditLinks
	addresses *GetLinkAddress
}

// NewReplaceLinks writes the links between two notes through the editor of a
// note's relationships.
func NewReplaceLinks(linking EditLinks, addresses *GetLinkAddress) ReplaceLinks {
	return ReplaceLinks{linking: linking, addresses: addresses}
}

// Execute leaves each of the two notes holding the link given for it to the
// other, or none. Every link is checked before any note is written, and each
// note is read and written once.
func (u ReplaceLinks) Execute(
	ctx context.Context, v domain.Vault, first, second string, links []PairLink,
) error {
	content, err := checkPairLinks(first, second, links)
	if err != nil {
		return err
	}
	kind := ""
	for _, link := range links {
		kind = link.Type
	}
	ends := [2]struct{ note, other string }{{first, second}, {second, first}}
	for _, end := range ends {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := u.replaceIn(ctx, v, end.note, end.other, kind, content[end.note]); err != nil {
			return fmt.Errorf("replace the link in %s: %w", end.note, err)
		}
	}
	return nil
}

// replaceIn leaves one note holding the link given to the other, or none. A note
// given none loses its records of that type to the other note.
func (u ReplaceLinks) replaceIn(
	ctx context.Context, v domain.Vault, from, to, kind string, link *PairLink,
) error {
	target, err := u.addresses.Execute(ctx, v, to)
	if err != nil {
		return err
	}
	_, err = u.linking.newEdit().Apply(ctx, v, from, func(doc *markdown.Document) error {
		if link == nil {
			_, err := doc.RemoveLink(domain.LinkKey{Target: target, Type: kind})
			return err
		}
		return doc.AddLink(domain.Link{
			Target: target, Role: link.Role, Type: link.Type, Why: link.Why,
			Label: link.Label, Marker: link.Marker,
		})
	})
	return err
}

// checkPairLinks returns the link each note holds, and rejects links that do
// not run between the two notes, repeat an end, or have an unknown role.
func checkPairLinks(first, second string, links []PairLink) (map[string]*PairLink, error) {
	if first == "" || second == "" || first == second {
		return nil, fmt.Errorf("%w: %q and %q", ErrNotAPair, first, second)
	}
	content := map[string]*PairLink{}
	for i := range links {
		link := &links[i]
		isForward := link.From == first && link.To == second
		isBackward := link.From == second && link.To == first
		if !isForward && !isBackward {
			return nil, fmt.Errorf("%w: %s to %s", ErrNotAPair, link.From, link.To)
		}
		if !domain.IsKnownRole(link.Role) {
			return nil, fmt.Errorf("%q is not a role a link can carry", link.Role)
		}
		if !domain.IsWritableMarker(link.Marker) {
			return nil, fmt.Errorf("%q is not a marker a link can carry", link.Marker)
		}
		if content[link.From] != nil {
			return nil, fmt.Errorf("%w: two links written in %s", ErrNotAPair, link.From)
		}
		content[link.From] = link
	}
	return content, nil
}
