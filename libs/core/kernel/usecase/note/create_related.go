package note

import (
	"context"
	"errors"
	"fmt"
	pathpkg "path"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// UntitledName is the title of a note before the person has named it.
const UntitledName = "Untitled note"

// MaxUntitledAttempts is the number of numbered titles tried before the vault is
// reported to hold all of them.
const MaxUntitledAttempts = 100

// CreateInRelation creates a note joined to an existing one. The new note writes
// the record about From, with the inverse role, and is filed in the folder of From.
type CreateInRelation struct {
	create noteCreating
	names  edit.NameQueries
}

// noteCreating is the creation of a note that CreateInRelation joins to another.
type noteCreating interface {
	Execute(ctx context.Context, v domain.Vault, in NewNote) (CreateResult, error)
	ExecuteUntitled(ctx context.Context, v domain.Vault, in NewNote) (CreateResult, error)
}

// NewCreateInRelation builds the use case over the creation of a note and the
// address under which a link names the note it is related to.
func NewCreateInRelation(create noteCreating, names edit.NameQueries) CreateInRelation {
	return CreateInRelation{create: create, names: names}
}

// Execute creates the note. An untitled note takes the lowest free number; a
// title that is already taken fails with vault.ErrAlreadyExists.
func (u CreateInRelation) Execute(ctx context.Context, v domain.Vault, in domain.RelatedNote) (CreateResult, error) {
	if in.From == "" || !domain.IsConnectionRole(in.Role) {
		return CreateResult{}, fmt.Errorf("%w: %q of %q", domain.ErrNotCreatable, in.Role, in.From)
	}
	inverse, _ := domain.GetInverseRole(in.Role)
	target, err := getLinkAddress(ctx, u.names, v, in.From)
	if err != nil {
		return CreateResult{}, err
	}
	folder := pathpkg.Dir(in.From)
	if folder == "." {
		folder = ""
	}
	links := []domain.Link{{Target: target, Role: inverse, Type: in.Type, Marker: in.Marker}}

	if in.Title != "" {
		return u.create.Execute(ctx, v, NewNote{Title: in.Title, Path: folder, Links: links})
	}
	return u.create.ExecuteUntitled(ctx, v, NewNote{Path: folder, Links: links})
}

// ExecuteUntitled creates a note under the lowest free "Untitled note N". The
// title of the input is ignored. It fails with domain.ErrNoFreeName when every
// numbered name is taken.
func (u Create) ExecuteUntitled(ctx context.Context, v domain.Vault, in NewNote) (CreateResult, error) {
	for attempt := 1; attempt <= MaxUntitledAttempts; attempt++ {
		in.Title = getUntitledName(attempt)
		made, err := u.Execute(ctx, v, in)
		if errors.Is(err, vault.ErrAlreadyExists) {
			continue
		}
		return made, err
	}
	return CreateResult{}, domain.ErrNoFreeName
}

func getUntitledName(attempt int) string {
	if attempt == 1 {
		return UntitledName
	}
	return fmt.Sprintf("%s %d", UntitledName, attempt)
}
