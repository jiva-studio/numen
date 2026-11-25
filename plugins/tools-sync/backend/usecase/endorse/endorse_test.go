package endorse_test

import (
	"bytes"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/healthkey"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/chaincourier"
)

const (
	laptop domain.VaultID = "Laptop vault"
	phone  domain.VaultID = "Phone vault"
	tablet domain.VaultID = "Tablet vault"
	list                  = "Shopping list.md"
)

// account is a person with a laptop and a phone that share one synced vault, where the phone wrote the shopping list.
type account struct {
	server *synctest.Server
	clock  *synctest.Clock
	words  []string
	laptop *synctest.Computer
	phone  *synctest.Computer
}

func newAccount(t *testing.T) *account {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	a := &account{server: server, clock: clock, words: words}
	a.laptop = synctest.NewFirstComputer(t, server, clock, string(laptop), words)
	a.laptop.Connect(t, laptop)
	a.phone = synctest.NewSecondComputer(t, server, clock, string(phone), words)
	a.phone.ConnectExisting(t, phone, a.laptop.Link(t, laptop).RemoteVaultID)
	a.phone.WriteNote(phone, list, "milk")
	if err := a.phone.Sync(t.Context(), phone); err != nil {
		t.Fatal(err)
	}
	if err := a.laptop.Sync(t.Context(), laptop); err != nil {
		t.Fatal(err)
	}
	if text, isThere := a.laptop.ReadNote(laptop, list); !isThere || text != "milk" {
		t.Fatalf("the laptop reads %q (%v) before the revoke, want what the phone wrote", text, isThere)
	}
	if _, err := chaincourier.NewAdopt(a.laptop.ChainDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	return a
}

func (a *account) revokePhone(t *testing.T) {
	t.Helper()
	if err := a.laptop.Fake.RevokeDevice(a.phone.Fake.Public().DeviceSignKey); err != nil {
		t.Fatal(err)
	}
}

func (a *account) publishWithoutEndorsing(t *testing.T) {
	t.Helper()
	pending, err := a.laptop.Client.GetChainPublish(t.Context())
	if err != nil || !pending.IsPending {
		t.Fatalf("GetChainPublish() = %+v, %v, want the revoke pending", pending, err)
	}
	if _, err := a.laptop.Cloud.AppendChain(t.Context(), pending.Chain); err != nil {
		t.Fatal(err)
	}
	if err := a.laptop.Client.PutChainPublished(t.Context(), true, nil); err != nil {
		t.Fatal(err)
	}
}

func (a *account) publish(t *testing.T) {
	t.Helper()
	if published, err := chaincourier.NewPublish(a.laptop.ChainDeps()).Execute(t.Context()); err != nil || !published {
		t.Fatalf("published = %v, error = %v, want the revoke published", published, err)
	}
}

func (a *account) heads(t *testing.T) []*syncv1.StoredRevision {
	t.Helper()
	remote := a.laptop.GetRemoteID(t, laptop)
	return a.laptop.ReadChanges(t, remote)
}

func TestPublish_ARevokeWhileAnotherDeviceKeepsWorkingEndorsesEveryHeadTheRevokedDeviceWrote(t *testing.T) {
	a := newAccount(t)
	phoneKey := a.phone.Fake.Public().DeviceSignKey
	laptopKey := a.laptop.Fake.Public().DeviceSignKey
	before := a.heads(t)
	if len(before) == 0 || !bytes.Equal(before[0].GetRevision().GetAuthorDeviceSignKey(), phoneKey) {
		t.Fatalf("the heads before the revoke = %d, first by the phone = %v, want the head of the list by the phone", len(before), len(before) > 0)
	}
	a.revokePhone(t)

	a.publish(t)

	for _, head := range a.heads(t) {
		revision := head.GetRevision()
		if !bytes.Equal(revision.GetAuthorDeviceSignKey(), laptopKey) {
			t.Errorf("a head is written by %x, want the laptop", revision.GetAuthorDeviceSignKey())
		}
		old := before[0].GetRevision()
		if revision.GetFileVersion() != old.GetFileVersion()+1 || !bytes.Equal(revision.GetBlobId(), old.GetBlobId()) ||
			!bytes.Equal(revision.GetParentRev(), before[0].GetRevId()) {
			t.Errorf("the endorsement is version %d of blob %x over %x, want version %d of the same blob over the head of the phone",
				revision.GetFileVersion(), revision.GetBlobId(), revision.GetParentRev(), old.GetFileVersion()+1)
		}
	}
}

func TestPublish_ADeviceThatJoinsAfterTheRevokeStillReadsTheNotesTheRevokedPhoneWrote(t *testing.T) {
	a := newAccount(t)
	a.revokePhone(t)
	a.publish(t)
	late := synctest.NewSecondComputer(t, a.server, a.clock, string(tablet), a.words)
	late.ConnectExisting(t, tablet, a.laptop.Link(t, laptop).RemoteVaultID)

	if err := late.Sync(t.Context(), tablet); err != nil {
		t.Fatal(err)
	}

	if text, isThere := late.ReadNote(tablet, list); !isThere || text != "milk" {
		t.Errorf("the new device reads %q (%v), want the list of the revoked phone", text, isThere)
	}
	if late.HasItemKey(healthkey.Create("unverified-changes", string(tablet))) {
		t.Error("the new device says some changes could not be verified")
	}
}

func TestPublish_TheLaptopKeepsWorkingOnTheListAfterThePhoneIsRevoked(t *testing.T) {
	a := newAccount(t)
	a.revokePhone(t)
	a.publish(t)

	a.laptop.WriteNote(laptop, list, "milk and eggs")
	if err := a.laptop.Sync(t.Context(), laptop); err != nil {
		t.Fatal(err)
	}
	late := synctest.NewSecondComputer(t, a.server, a.clock, string(tablet), a.words)
	late.ConnectExisting(t, tablet, a.laptop.Link(t, laptop).RemoteVaultID)
	if err := late.Sync(t.Context(), tablet); err != nil {
		t.Fatal(err)
	}

	if text, _ := late.ReadNote(tablet, list); text != "milk and eggs" {
		t.Errorf("the new device reads %q, want the edit of the laptop", text)
	}
}

func TestPublish_ARevokedDevicesUnendorsedNoteIsNotTaken(t *testing.T) {
	a := newAccount(t)
	a.revokePhone(t)
	a.publishWithoutEndorsing(t)
	late := synctest.NewSecondComputer(t, a.server, a.clock, string(tablet), a.words)
	late.ConnectExisting(t, tablet, a.laptop.Link(t, laptop).RemoteVaultID)
	if err := late.Sync(t.Context(), tablet); err != nil {
		t.Fatal(err)
	}

	if text, isThere := late.ReadNote(tablet, list); isThere {
		t.Errorf("the new device downloaded %q written by the revoked phone that nobody endorsed", text)
	}
	if !late.HasItemKey(healthkey.Create("unverified-changes", string(tablet))) {
		t.Error("the new device does not say that changes could not be verified")
	}
}
