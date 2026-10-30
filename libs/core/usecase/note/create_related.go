package note

import (
	"context"
	"errors"
	"fmt"
	pathpkg "path"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// UntitledName is the title of a note before the person has named it.
const UntitledName = "Untitled note"

// MaxUntitledAttempts is the number of numbered titles tried before the vault is
// reported to hold all of them.
const MaxUntitledAttempts = 100

// CreateInRelation creates a note joined to an existing one. The new note writes
// the record about From, with the inverse role, and is filed in the folder of From.
type CreateInRelation struct {
	create    Create
	addresses *GetLinkAddress
}

// NewCreateInRelation builds the use case over the creation of a note and the
// address under which a link names the note it is related to.
func NewCreateInRelation(create Create, addresses *GetLinkAddress) CreateInRelation {
	return CreateInRelation{create: create, addresses: addresses}
}

// Execute creates the note. An untitled note takes the lowest free number; a
// title that is already taken fails with vaultport.ErrAlreadyExists.
func (u CreateInRelation) Execute(ctx context.Context, v domain.Vault, in domain.RelatedNote) (CreateResult, error) {
	if in.From == "" || !domain.IsConnectionRole(in.Role) {
		return CreateResult{}, fmt.Errorf("%w: %q of %q", domain.ErrNotCreatable, in.Role, in.From)
	}
	inverse, _ := domain.GetInverseRole(in.Role)
	target, err := u.addresses.Execute(ctx, v, in.From)
	if err != nil {
		return CreateResult{}, err
	}
	folder := pathpkg.Dir(in.From)
	if folder == "." {
		folder = ""
	}
	links := []domain.Link{{Target: target, Role: inverse, Type: in.Type}}

	if in.Title != "" {
		return u.create.Execute(ctx, v, NewNote{Title: in.Title, Path: folder, Links: links})
	}
	for attempt := 1; attempt <= MaxUntitledAttempts; attempt++ {
		made, err := u.create.Execute(ctx, v, NewNote{Title: getUntitledName(attempt), Path: folder, Links: links})
		if errors.Is(err, vaultport.ErrAlreadyExists) {
			continue
		}
		return made, err
	}
	return CreateResult{}, domain.ErrNoFreeName
}

// getUntitledName is the title tried on the given attempt, counted from one.
func getUntitledName(attempt int) string {
	if attempt == 1 {
		return UntitledName
	}
	return fmt.Sprintf("%s %d", UntitledName, attempt)
}
