package license

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
)

// Watch streams the full list of licenses when the stream opens and again whenever a license or
// the set of plugins changes.
type Watch struct {
	operation
}

// NewWatch returns the use case that watches license changes.
func NewWatch(deps Deps) *Watch {
	return &Watch{operation: deps.base}
}

// Execute sends the current list of licenses immediately and again whenever a change signal
// arrives, until ctx ends or send fails.
func (u *Watch) Execute(ctx context.Context, send func([]license.Info) error) error {
	list := NewList(Deps{base: u.operation})
	entries, err := list.Execute(ctx)
	if err != nil {
		return err
	}
	if err := send(entries); err != nil {
		return err
	}
	if u.changes == nil {
		<-ctx.Done()
		return nil
	}
	changes, stop := u.changes.Subscribe()
	defer stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case _, open := <-changes:
			if !open {
				return nil
			}
			entries, err := list.Execute(ctx)
			if err != nil {
				return err
			}
			if err := send(entries); err != nil {
				return err
			}
		}
	}
}
