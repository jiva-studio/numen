// Package links lists the links of notes and writes links into their frontmatter.
package links

import (
	"context"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/ports"
)

// LinkList is the outgoing links and the backlinks of a note.
type LinkList struct {
	Path      string
	Links     []domain.ResolvedLink
	Backlinks []domain.ResolvedLink
}

// UseCase works the links of the notes of one vault.
type UseCase struct {
	lookup   ports.LinkLookup
	editor   ports.LinkEditor
	resolver ports.NotePathResolver
}

// New creates a UseCase over the links of one vault; resolver finds the note an agent names.
func New(lookup ports.LinkLookup, editor ports.LinkEditor, resolver ports.NotePathResolver) *UseCase {
	return &UseCase{lookup: lookup, editor: editor, resolver: resolver}
}

// List returns the outgoing links and the backlinks of a note.
func (u *UseCase) List(ctx context.Context, name string) (LinkList, error) {
	if strings.TrimSpace(name) == "" {
		return LinkList{}, errors.New("path is required")
	}
	path := u.resolver.Resolve(ctx, name)
	outgoing, err := u.lookup.GetLinks(ctx, path)
	if err != nil {
		return LinkList{}, err
	}
	incoming, err := u.lookup.GetBacklinks(ctx, path)
	if err != nil {
		return LinkList{}, err
	}
	return LinkList{Path: path, Links: outgoing, Backlinks: incoming}, nil
}
