package webui

import "github.com/jiva-studio/numen/modules/libs/core/domain"

// changed is what a client is told: the files that are different now, notes
// and assets alike, or that the vault has to be read again.
type changed struct {
	paths  []string
	reload bool
	// renamed is the notes that are no longer where they were, each by where it
	// was and where it now is.
	renamed []domain.Move
}

// following is everyone listening for what moved.
//
// A listener that fell behind is told to read everything again, which is the
// same answer the watcher gives when more arrives at once than it can follow.
// A message that never arrives leaves a client showing something stale and
// certain it is current.
func following() audience[changed] {
	return audience[changed]{
		fallback: func(changed) changed { return changed{reload: true} },
		room:     8,
	}
}

// focusing is everyone drawing the vault, for the places something asks to be
// put in front of the person. A place asked for while a listener is busy
// replaces the one it has not read: what matters is the last place asked for.
func focusing() audience[domain.Place] {
	return audience[domain.Place]{latest: true, room: 1}
}

// drawing is everyone drawing the vault, for a change to a note being made
// while they are showing it.
//
// Each report carries the whole of what one change is doing, so a report
// replaces the one waiting about that change and nothing else: several changes
// are made at once, and each is drawn in a place of its own. The report that
// ends a change is replaced by nothing, and arrives whatever a listener is
// doing.
func drawing() audience[domain.Edit] {
	return audience[domain.Edit]{
		latest: true,
		about:  func(said domain.Edit) string { return said.Change },
		keep:   func(said domain.Edit) bool { return said.Done },
		room:   8,
	}
}
