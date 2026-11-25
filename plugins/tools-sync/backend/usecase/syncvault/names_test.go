package syncvault_test

import (
	"testing"

	"golang.org/x/text/unicode/norm"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

var (
	composedName   = norm.NFC.String("Йога и ёж.md")
	decomposedName = norm.NFD.String("Йога и ёж.md")
)

func TestSyncVault_ANoteNamedInDecomposedFormIsSentOnceAndNotAgain(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, decomposedName, "one")
	sync(t, p.a, source)
	reserved := p.countCalls(syncv1connect.BlobServiceReserveBlobsProcedure)
	committed := p.countCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure)

	sync(t, p.a, source)
	sync(t, p.a, source)

	if got := p.countCalls(syncv1connect.BlobServiceReserveBlobsProcedure); got != reserved {
		t.Errorf("reservations grew from %d to %d, want none for a note the server already holds", reserved, got)
	}
	if got := p.countCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure); got != committed {
		t.Errorf("commits grew from %d to %d, want none for a note the server already holds", committed, got)
	}
	if got := len(p.getLiveHeads(t)); got != 1 {
		t.Errorf("the server holds %d notes, want 1", got)
	}
}

func TestSyncVault_ADecomposedNoteChangedLaterIsSentAsOneEdit(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, decomposedName, "one")
	sync(t, p.a, source)
	p.a.WriteNote(source, decomposedName, "two")

	sync(t, p.a, source)
	sync(t, p.b, target)

	if got := len(p.getLiveHeads(t)); got != 1 {
		t.Errorf("the server holds %d notes, want 1", got)
	}
	wantNote(t, p.b, target, composedName, "two")
}

func TestSyncVault_ANoteReceivedUnderTheComposedNameLandsOnTheDecomposedFile(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.b.WriteNote(target, decomposedName, "one")
	sync(t, p.b, target)
	sync(t, p.a, source)
	p.a.WriteNote(source, composedName, "two")
	sync(t, p.a, source)

	sync(t, p.b, target)

	wantNote(t, p.b, target, decomposedName, "two")
	wantNoFile(t, p.b, target, composedName)
	sync(t, p.b, target)
	if got := len(p.getLiveHeads(t)); got != 1 {
		t.Errorf("the server holds %d notes, want 1", got)
	}
}
