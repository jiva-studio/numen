package editor

import "github.com/jiva-studio/numen/modules/libs/notes/domain"

// change is what a client is told: the files that are different now, notes
// and assets alike, or that the vault has to be read again.
type change struct {
	paths        []string
	shouldReload bool
	// renamed is the notes that are no longer where they were, each by where it
	// was and where it now is.
	renamed []domain.Move
}

// newChangeBroadcaster delivers what moved to its subscribers.
//
// A listener that fell behind is told to read everything again, which is the
// same answer the watcher gives when more arrives at once than it can follow.
// A message that never arrives leaves a client showing something stale and
// certain it is current.
func newChangeBroadcaster() broadcaster[change] {
	return broadcaster[change]{
		fallback: func(change) change { return change{shouldReload: true} },
		capacity: 8,
	}
}

// newPlaceBroadcaster delivers to every subscriber, for the places something asks to be
// put in front of the person. A place asked for while a listener is busy
// replaces the one it has not read: what matters is the last place asked for.
func newPlaceBroadcaster() broadcaster[domain.Place] {
	return broadcaster[domain.Place]{isLatest: true, capacity: 1}
}

// newEditBroadcaster delivers to every subscriber, for a change to a note being made
// while they are showing it.
//
// Each report carries the whole of what one change is doing, so a report
// replaces the one waiting about that change and nothing else: several changes
// are made at once, and each is drawn in a place of its own. The report that
// ends a change is replaced by nothing, and arrives whatever a listener is
// doing.
func newEditBroadcaster() broadcaster[domain.Edit] {
	return broadcaster[domain.Edit]{
		isLatest:   true,
		getTopic:   func(said domain.Edit) string { return said.Change },
		isRequired: func(said domain.Edit) bool { return said.IsDone },
		capacity:   8,
	}
}
