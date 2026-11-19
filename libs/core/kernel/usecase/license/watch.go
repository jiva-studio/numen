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
	changes, stop := u.subscribe()
	if stop != nil {
		defer stop()
	}
	list := NewList(Deps{base: u.operation})
	if err := sendLicenseList(ctx, list, send); err != nil {
		return err
	}
	return u.waitForChanges(ctx, changes, list, send)
}

func (u *Watch) subscribe() (<-chan struct{}, func()) {
	if u.changes == nil {
		return nil, nil
	}
	return u.changes.Subscribe()
}

func sendLicenseList(ctx context.Context, list *List, send func([]license.Info) error) error {
	entries, err := list.Execute(ctx)
	if err != nil {
		return err
	}
	return send(entries)
}

func (u *Watch) waitForChanges(
	ctx context.Context,
	changes <-chan struct{},
	list *List,
	send func([]license.Info) error,
) error {
	if changes == nil {
		<-ctx.Done()
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case _, open := <-changes:
			if !open {
				return nil
			}
			if err := sendLicenseList(ctx, list, send); err != nil {
				return err
			}
		}
	}
}
