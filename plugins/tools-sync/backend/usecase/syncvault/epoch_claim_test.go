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
