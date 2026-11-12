package editor

import (
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
	return n.ChangeEvents.Subscribe()
}

// SubscribePlaces delivers each place the window is asked to show until the returned function is called.
func (n *Notices) SubscribePlaces() (<-chan domain.Place, func()) { return n.Places.Subscribe() }

// SubscribeTabIDs delivers the id of each tab the window is asked to bring to the front until the
// returned function is called.
func (n *Notices) SubscribeTabIDs() (<-chan string, func()) { return n.TabIDs.Subscribe() }

// SubscribeEdits returns the edits being made to notes and the function that ends the subscription.
func (n *Notices) SubscribeEdits() (<-chan domain.Edit, func()) { return n.Edits.Subscribe() }

// PublishChange tells the views drawing the vault that the paths changed.
func (n *Notices) PublishChange(paths ...string) {
	n.ChangeEvents.Publish(window.Change{Paths: paths})
}

type broadcaster[T any] struct {
	inner broadcast.Broadcaster[T]
}

func newChangeBroadcaster() broadcaster[window.Change] {
	return broadcaster[window.Change]{
		inner: broadcast.Broadcaster[window.Change]{
			Fallback: func(window.Change) window.Change { return window.Change{ShouldReload: true} },
			Capacity: 8,
		},
	}
}

func newPlaceBroadcaster() broadcaster[domain.Place] {
	return broadcaster[domain.Place]{
		inner: broadcast.Broadcaster[domain.Place]{IsLatest: true, Capacity: 1},
	}
}

func newTabIDBroadcaster() broadcaster[string] {
	return broadcaster[string]{
		inner: broadcast.Broadcaster[string]{IsLatest: true, Capacity: 1},
	}
}

func newEditBroadcaster() broadcaster[domain.Edit] {
	return broadcaster[domain.Edit]{
		inner: broadcast.Broadcaster[domain.Edit]{
			IsLatest:   true,
			GetTopic:   func(response domain.Edit) string { return response.Change },
			IsRequired: func(reply domain.Edit) bool { return reply.IsDone },
			Capacity:   8,
		},
	}
}

// Publish sends what to every subscriber.
func (a *broadcaster[T]) Publish(what T) { a.inner.Publish(what) }

// Subscribe returns what is published from now on and the function that ends the subscription.
func (a *broadcaster[T]) Subscribe() (<-chan T, func()) { return a.inner.Subscribe() }
