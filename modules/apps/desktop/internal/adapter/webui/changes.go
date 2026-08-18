package webui

import "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"

// changed is what a client is told: the notes that are different now, or that
// the vault has to be read again.
type changed struct {
	paths  []string
	reload bool
}

// following is everyone listening for what moved.
//
// A listener that fell behind is told to read everything again, which is the
// same answer the watcher gives when more arrives at once than it can follow.
// A message that never arrives leaves a client showing something stale and
// certain it is current.
func following() audience[changed] {
	return audience[changed]{
		behind: func(changed) changed { return changed{reload: true} },
		room:   8,
	}
}

// focusing is everyone drawing the vault, for the notes something asks to be
// put in front of the person. A note asked for while a listener is busy
// replaces the one it has not read: what matters is the last note asked for.
func focusing() audience[string] {
	return audience[string]{latest: true, room: 1}
}

// drawing is everyone drawing the vault, for a change to a note being made
// while they are showing it.
//
// A report arriving while a listener is busy replaces the one it has not read:
// each report carries the whole of what a change is doing, so the newest is the
// only one worth having. The report that ends a change is not one of those, and
// arrives whatever a listener is doing.
func drawing() audience[domain.Editing] {
	return audience[domain.Editing]{
		latest: true,
		keep:   func(said domain.Editing) bool { return said.Done },
		room:   1,
	}
}
