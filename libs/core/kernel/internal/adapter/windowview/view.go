// Package windowview carries what the core asks a window to show: a place to focus, a note that
// moved, a write in progress, and a tab to bring forward.
package windowview

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// Notices takes what the window is asked to show. A request made while no client is subscribed is
// not shown.
type Notices interface {
	PublishPlace(pos domain.Place)
	PublishMove(went domain.Move)
	PublishEdit(edit domain.Edit)
	PublishTab(id string)
}

// View forwards the requests of the core to the clients subscribed to the window.
type View struct{ Notices Notices }

// FocusPlace asks the clients to show the place.
func (v View) FocusPlace(_ context.Context, pos domain.Place) error {
	v.Notices.PublishPlace(pos)
	return nil
}

// ShowMove tells the clients that a note moved.
func (v View) ShowMove(_ context.Context, went domain.Move) error {
	v.Notices.PublishMove(went)
	return nil
}

// ShowEdit tells the clients that a change to a note is in progress.
func (v View) ShowEdit(_ context.Context, edit domain.Edit) error {
	v.Notices.PublishEdit(edit)
	return nil
}

// ActivateTab asks the clients to bring the tab forward.
func (v View) ActivateTab(_ context.Context, id string) error {
	v.Notices.PublishTab(id)
	return nil
}
