package syncvault_test

import (
	"bytes"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

const textEpochMismatch = "The sync server reported a restore that its signed announcement does not confirm, so nothing was changed"

func fakeEpochID() []byte { return bytes.Repeat([]byte{0xAB}, 16) }

func TestSyncVault_IgnoresARestoreEpochIdWhenTheServerNeverAnnouncedARestore(t *testing.T) {
	p := newPair(t)
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	before := p.b.Link(t, target)

	err := p.b.NewSyncVaultOver(synctest.EpochClaim{Revisions: p.b.Cloud, ID: fakeEpochID()}).Execute(t.Context(), target)

	if err != nil {
		t.Fatalf("pass = %v, want the pass to end with a warning", err)
	}
	if after := p.b.Link(t, target); !bytes.Equal(after.RestoreEpochID, before.RestoreEpochID) {
		t.Errorf("epoch id of the link = %x, want it kept", after.RestoreEpochID)
	}
	if !p.b.HasItem(textEpochMismatch) {
		t.Error("no warning says the restore of the server is not confirmed")
	}
}

func TestSyncVault_ClearsTheWarningOfAnUnconfirmedRestoreWhenTheServerAnswersHonestlyAgain(t *testing.T) {
	p := newPair(t)
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	if err := p.b.NewSyncVaultOver(synctest.EpochClaim{Revisions: p.b.Cloud, ID: fakeEpochID()}).Execute(t.Context(), target); err != nil {
		t.Fatal(err)
	}

	sync(t, p.b, target)

	if p.b.HasItem(textEpochMismatch) {
		t.Error("the warning stays after an honest answer")
	}
	wantNote(t, p.b, target, "Shopping list.md", "milk")
}

func TestSyncVault_IgnoresARestoreEpochIdThatTheSignedAnnouncementDoesNotName(t *testing.T) {
	p := newOwnPair(t)
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	sync(t, p.b, target)
	announce(t, p)
	sync(t, p.b, target)
	before := p.b.Link(t, target)
	commits := countCommits(p)
	p.b.WriteNote(target, "Plan.md", "tea")

	err := p.b.NewSyncVaultOver(synctest.EpochClaim{Revisions: p.b.Cloud, ID: fakeEpochID()}).Execute(t.Context(), target)

	if err != nil {
		t.Fatalf("pass = %v, want the pass to end with a warning", err)
	}
	after := p.b.Link(t, target)
	if !bytes.Equal(after.RestoreEpochID, before.RestoreEpochID) {
		t.Errorf("epoch id of the link = %x, want %x kept: nobody signed the claimed one", after.RestoreEpochID, before.RestoreEpochID)
	}
	if after.ServerSeq != before.ServerSeq || after.ServerSeq == 0 {
		t.Errorf("position = %d, want %d kept: no recovery may reset it", after.ServerSeq, before.ServerSeq)
	}
	if got := countCommits(p); got != commits {
		t.Errorf("commits grew from %d to %d, want nothing sent for a restore nobody confirmed", commits, got)
	}
	if !p.b.HasItem(textEpochMismatch) {
		t.Error("no warning says the restore of the server is not confirmed")
	}
}
