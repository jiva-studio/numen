package editor

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/broadcast"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Notices is what the window is asked to show: the changes of the vault, the places to focus, the edits
// in progress and the tabs to bring forward. A request made while no client is subscribed is not shown.
type Notices struct {
	ChangeEvents broadcaster[window.Change]
	Places       broadcaster[domain.Place]
	TabIDs       broadcaster[string]
	Edits        broadcaster[domain.Edit]
}

func newNotices() *Notices {
	return &Notices{
		ChangeEvents: newChangeBroadcaster(),
		Places:       newPlaceBroadcaster(),
		TabIDs:       newTabIDBroadcaster(),
		Edits:        newEditBroadcaster(),
	}
}

// SubscribeChanges delivers each change of the vault until the returned function is called.
func (n *Notices) SubscribeChanges() (<-chan window.Change, func()) {
	return n.ChangeEvents.subscribe()
}

// SubscribePlaces delivers each place the window is asked to show until the returned function is called.
func (n *Notices) SubscribePlaces() (<-chan domain.Place, func()) { return n.Places.subscribe() }

// SubscribeTabIDs delivers the id of each tab the window is asked to bring to the front until the
// returned function is called.
func (n *Notices) SubscribeTabIDs() (<-chan string, func()) { return n.TabIDs.subscribe() }

// SubscribeEdits returns the edits being made to notes and the function that ends the subscription.
func (n *Notices) SubscribeEdits() (<-chan domain.Edit, func()) { return n.Edits.subscribe() }

// PublishChange tells the views drawing the vault that the paths changed.
func (n *Notices) PublishChange(paths ...string) {
	n.ChangeEvents.publish(window.Change{Paths: paths})
}

// FocusPlace asks the clients to show the place.
func (n *Notices) FocusPlace(_ context.Context, pos domain.Place) error {
	n.Places.publish(pos)
	return nil
}

// ShowMove tells the clients that a note moved.
func (n *Notices) ShowMove(_ context.Context, went domain.Move) error {
	n.ChangeEvents.publish(window.Change{Renamed: []domain.Move{went}})
	return nil
}

// ShowEdit tells the clients that a change to a note is in progress.
func (n *Notices) ShowEdit(_ context.Context, edit domain.Edit) error {
	n.Edits.publish(edit)
	return nil
}

// ActivateTab asks the clients to bring the tab forward.
func (n *Notices) ActivateTab(_ context.Context, id string) error {
	n.TabIDs.publish(id)
	return nil
}

type broadcaster[T any] struct {
	broadcast.Broadcaster[T]
}

func (a *broadcaster[T]) subscribe() (<-chan T, func()) { return a.Broadcaster.Subscribe() }
func (a *broadcaster[T]) publish(what T)                { a.Broadcaster.Publish(what) }

func newChangeBroadcaster() broadcaster[window.Change] {
	return broadcaster[window.Change]{
		Broadcaster: broadcast.Broadcaster[window.Change]{
			Fallback: func(window.Change) window.Change { return window.Change{ShouldReload: true} },
			Capacity: 8,
		},
	}
}

func newPlaceBroadcaster() broadcaster[domain.Place] {
	return broadcaster[domain.Place]{
		Broadcaster: broadcast.Broadcaster[domain.Place]{IsLatest: true, Capacity: 1},
	}
}

func newTabIDBroadcaster() broadcaster[string] {
	return broadcaster[string]{
		Broadcaster: broadcast.Broadcaster[string]{IsLatest: true, Capacity: 1},
	}
}

func newEditBroadcaster() broadcaster[domain.Edit] {
	return broadcaster[domain.Edit]{
		Broadcaster: broadcast.Broadcaster[domain.Edit]{
			IsLatest:   true,
			GetTopic:   func(response domain.Edit) string { return response.Change },
			IsRequired: func(reply domain.Edit) bool { return reply.IsDone },
			Capacity:   8,
		},
	}
}
