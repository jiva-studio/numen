package license

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
)

// List returns every license declared by the installed plugins together with its activation state
// on this machine.
type List struct {
	operation
}

// NewList returns the use case that lists all declared licenses.
func NewList(deps Deps) *List {
	return &List{operation: deps.base}
}

// Execute returns the licenses in the order of their merged declarations, and schedules background
// revalidation for any stored license whose verification TTL has elapsed.
func (u *List) Execute(ctx context.Context) ([]license.Info, error) {
	u.gate.Lock()
	decls := u.listDeclarations()
	infos := make([]license.Info, 0, len(decls))
	var staleIDs []string
	for _, decl := range decls {
		info, needsValidation, err := u.buildInfo(ctx, decl)
		if err != nil {
			u.gate.Unlock()
			return nil, err
		}
		if needsValidation {
			staleIDs = append(staleIDs, decl.ID)
		}
		infos = append(infos, info)
	}
	u.gate.Unlock()

	if len(staleIDs) > 0 && u.validator != nil {
		bgCtx := context.WithoutCancel(ctx)
		go u.revalidateInBackground(bgCtx, staleIDs)
	}
	return infos, nil
}

func (u *List) revalidateInBackground(ctx context.Context, licenseIDs []string) {
	verify := NewVerify(Deps{base: u.operation})
	for _, id := range licenseIDs {
		if _, err := verify.Execute(ctx, id); err != nil {
			continue
		}
	}
}
