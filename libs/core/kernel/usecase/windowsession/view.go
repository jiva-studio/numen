package windowsession

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// PlaceFront asks the clients to show a place.
type PlaceFront interface {
	Publish(pos domain.Place)
}

// EditFront tells the clients that a change to a note is in progress.
type EditFront interface {
	Publish(edit domain.Edit)
}

// View is what the core asks the window to show: a place to focus, a note that moved and a write in progress.
// A request made while no client is subscribed is not shown.
type View struct {
	places  PlaceFront
	edits   EditFront
	changes Changes
}

// NewView makes the view over the fronts the clients of the window subscribe to.
func NewView(places PlaceFront, edits EditFront, changes Changes) View {
	return View{places: places, edits: edits, changes: changes}
}

// FocusPlace asks the clients to show the place.
func (v View) FocusPlace(_ context.Context, pos domain.Place) error {
	v.places.Publish(pos)
	return nil
}

// ShowMove tells the clients that a note moved.
func (v View) ShowMove(_ context.Context, went domain.Move) error {
	v.changes.Publish(window.Change{Renamed: []domain.Move{went}})
	return nil
}

// ShowEdit tells the clients that a change to a note is in progress.
func (v View) ShowEdit(_ context.Context, edit domain.Edit) error {
	v.edits.Publish(edit)
	return nil
}
