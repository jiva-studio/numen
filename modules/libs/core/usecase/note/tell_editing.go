package note

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
)

// TellEditing is what a write says about itself while it is being made, for whoever
// is looking at the note. Nothing is said where nobody is drawing.
type TellEditing func(ctx context.Context, said domain.Edit)

// begins names one change and says what it is about to do. What comes back ends
// it, and ends it whether the change landed or was refused.
func (tell TellEditing) begins(ctx context.Context, said domain.Edit) func() {
	if tell == nil {
		return func() {}
	}
	name, err := ulid.New(time.Now())
	if err != nil {
		return func() {}
	}
	said.Change = name
	tell(ctx, said)
	return func() {
		tell(ctx, domain.Edit{Change: name, Path: said.Path, Done: true})
	}
}
