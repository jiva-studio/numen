package receive_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

const (
	owned  domain.VaultID = "Test"
	folder string         = "Food"
)

type world struct {
	server *synctest.Server
	clock  *synctest.Clock
	owner  *synctest.Computer
	reader *synctest.Computer
}

func newWorldOn(t *testing.T, cfg fakeserver.Config) *world {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, cfg)
	owner := synctest.NewFirstComputer(t, server, clock, string(owned), synctest.NewWords(t))
	if cfg.NeedsSubscription {
		setPhase(t, server, "active")
	}
	owner.Connect(t, owned)
	reader := synctest.NewReaderComputer(t, server, clock, "Reader")
	return &world{server: server, clock: clock, owner: owner, reader: reader}
}

func newWorld(t *testing.T) *world {
	t.Helper()
	return newWorldOn(t, fakeserver.Config{})
}

func setPhase(t *testing.T, server *synctest.Server, phase string) {
	t.Helper()
	if status := server.CallAdmin(t, "/admin/subscription?phase="+phase); status >= 300 {
		t.Fatalf("the admin path answered %d", status)
	}
}

func (w *world) syncOwner(t *testing.T) {
	t.Helper()
	if err := w.owner.Sync(t.Context(), owned); err != nil {
		t.Fatalf("pass of the owner: %v", err)
	}
}

func (w *world) shareWith(t *testing.T, reader *synctest.Computer, label string) synctest.ShareInvitation {
	t.Helper()
	state, err := w.owner.GetShare(t.Context(), owned, folder)
	if err != nil {
		t.Fatal(err)
	}
	if state.State == "notShared" {
		if err := w.owner.CreateShare(t.Context(), owned, folder); err != nil {
			t.Fatal(err)
		}
	}
	code, err := reader.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	invitation, err := w.owner.AddReader(t.Context(), owned, folder, code.Code, label, false)
	if err != nil {
		t.Fatal(err)
	}
	return invitation
}

func (w *world) receive(t *testing.T, reader *synctest.Computer, invitation synctest.ShareInvitation) synctest.ReceivedShare {
	t.Helper()
	received, err := reader.AcceptInvitation(t.Context(), invitation.Text)
	if err != nil {
		t.Fatal(err)
	}
	return received
}

func wantNote(t *testing.T, c *synctest.Computer, id domain.VaultID, path, text string) {
	t.Helper()
	if got, ok := c.ReadNote(id, path); !ok || got != text {
		t.Errorf("note %s = %q (there: %v), want %q", path, got, ok, text)
	}
}

func wantNoNote(t *testing.T, c *synctest.Computer, id domain.VaultID, path string) {
	t.Helper()
	if got, ok := c.ReadNote(id, path); ok {
		t.Errorf("note %s = %q, want no such note", path, got)
	}
}

func TestAcceptInvitation_MakesASeparateVaultWithTheNotesOfTheFolderAtItsRoot(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.owner.WriteNote(owned, "Food/Dinner/Stew.md", "beef")
	w.owner.WriteNote(owned, "Plans/Trip.md", "Lisbon")
	w.syncOwner(t)

	received := w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))

	wantNote(t, w.reader, received.VaultID, "Soup.md", "carrots")
	wantNote(t, w.reader, received.VaultID, "Dinner/Stew.md", "beef")
	wantNoNote(t, w.reader, received.VaultID, "Plans/Trip.md")
	wantNoNote(t, w.reader, received.VaultID, "Food/Soup.md")
	if received.Name != "Food" {
		t.Errorf("name = %q, want Food: the last segment of the folder", received.Name)
	}
	if _, ok := w.reader.ReadNote("Reader", "Plans/Trip.md"); ok {
		t.Error("the notes outside the folder arrived in the own vault of the reader")
	}
}

func TestAcceptInvitation_KeepsTheFolderInTheVaultOfTheOwner(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)

	w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))

	wantNote(t, w.owner, owned, "Food/Soup.md", "carrots")
}

func TestAcceptInvitation_RecordsTheReceivedKindAndThePinnedOwnerInTheLink(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)

	received := w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))

	link := w.reader.Link(t, received.VaultID)
	if link.Kind != vaultlink.KindReceived || link.Received == nil || len(link.Received.OwnerRoot) != 32 || len(link.Received.OwnerChain) == 0 {
		t.Errorf("link = %+v, want kind received with the pinned root and chain of the owner", link)
	}
	if got := w.reader.Vaults.GetManagedBy(received.VaultID); got != syncprotocol.SignInPluginID {
		t.Errorf("managed by %q, want the sync plugin", got)
	}
}

func TestAcceptInvitation_RefusesTheSameShareTwice(t *testing.T) {
	w := newWorld(t)
	invitation := w.shareWith(t, w.reader, "Anna")
	w.receive(t, w.reader, invitation)

	if _, err := w.reader.AcceptInvitation(t.Context(), invitation.Text); !errors.Is(err, share.ErrAlreadyReceived) {
		t.Errorf("error = %v, want ErrAlreadyReceived", err)
	}
	received, err := w.reader.ListReceived(t.Context())
	if err != nil || len(received) != 1 {
		t.Errorf("received = %v, error = %v, want the one share", received, err)
	}
}

func TestReadInvitation_RefusesAnInvitationOfAnotherPersonAndStoresNothing(t *testing.T) {
	w := newWorld(t)
	invitation := w.shareWith(t, w.reader, "Anna")
	stranger := synctest.NewReaderComputer(t, w.server, w.clock, "Stranger")
	before, err := stranger.Vaults.GetVaults(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	preview, previewErr := stranger.ReadInvitation(t.Context(), invitation.Text)
	_, acceptErr := stranger.AcceptInvitation(t.Context(), invitation.Text)

	if !errors.Is(previewErr, share.ErrNotForThisAccount) || preview.IsForThisAccount {
		t.Errorf("preview = %+v, error = %v, want ErrNotForThisAccount", preview, previewErr)
	}
	if !errors.Is(acceptErr, share.ErrNotForThisAccount) {
		t.Errorf("accept: error = %v, want ErrNotForThisAccount", acceptErr)
	}
	after, err := stranger.Vaults.GetVaults(t.Context())
	if err != nil || len(after) != len(before) {
		t.Errorf("vaults after = %d, before = %d (%v), want no vault added", len(after), len(before), err)
	}
	if received, err := stranger.ListReceived(t.Context()); err != nil || len(received) != 0 {
		t.Errorf("received = %v, error = %v, want nothing stored", received, err)
	}
}

func TestReadInvitation_ShowsTheNameOfTheShareAndTheFingerprintOfTheOwnerAndStoresNothing(t *testing.T) {
	w := newWorld(t)
	invitation := w.shareWith(t, w.reader, "Anna")
	ownerCode, err := w.owner.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	preview, err := w.reader.ReadInvitation(t.Context(), invitation.Text)
	if err != nil {
		t.Fatal(err)
	}

	if preview.ShareName != "Food" || preview.OwnerFingerprint != ownerCode.Fingerprint || !preview.IsForThisAccount || preview.AlreadyReceived {
		t.Errorf("preview = %+v, want Food, the fingerprint of the owner %s, for this account, not received", preview, ownerCode.Fingerprint)
	}
	if received, err := w.reader.ListReceived(t.Context()); err != nil || len(received) != 0 {
		t.Errorf("received = %v, error = %v, want nothing stored by a read", received, err)
	}
}

func TestAcceptInvitation_NumbersTheFolderOfASecondShareWithTheSameName(t *testing.T) {
	w := newWorld(t)
	other := synctest.NewFirstComputer(t, w.server, w.clock, "Other", synctest.NewWords(t))
	other.Connect(t, "Other")
	if err := other.CreateShare(t.Context(), "Other", folder); err != nil {
		t.Fatal(err)
	}
	code, err := w.reader.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	second, err := other.AddReader(t.Context(), "Other", folder, code.Code, "Anna", false)
	if err != nil {
		t.Fatal(err)
	}

	first := w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))
	next := w.receive(t, w.reader, second)

	if first.Name == next.Name || !strings.HasPrefix(next.Name, "Food") {
		t.Errorf("names %q and %q, want two distinct names that start with Food", first.Name, next.Name)
	}
}

func TestAcceptInvitation_RefusesAnInvitationOfTheSameShareWithAnotherOwnerRoot(t *testing.T) {
	w := newWorld(t)
	invitation := w.shareWith(t, w.reader, "Anna")
	parsed, err := share.ParseInvitation(invitation.Text)
	if err != nil {
		t.Fatal(err)
	}
	impostor := synctest.NewReaderComputer(t, w.server, w.clock, "Impostor")
	impostorCode, err := impostor.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	chain, err := share.ParseReaderCode(impostorCode.Code)
	if err != nil {
		t.Fatal(err)
	}
	w.receive(t, w.reader, invitation)
	parsed.OwnerChain = chain.Bytes()

	_, err = w.reader.AcceptInvitation(t.Context(), share.NewInvitation(parsed))

	if !errors.Is(err, share.ErrOwnerChanged) {
		t.Errorf("error = %v, want ErrOwnerChanged", err)
	}
	received, listErr := w.reader.ListReceived(t.Context())
	if listErr != nil || len(received) != 1 {
		t.Errorf("received = %v, error = %v, want the pinned share left as it was", received, listErr)
	}
}

func TestPullShare_PicksUpANoteChangedByTheOwnerAndANewNote(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	received := w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))

	w.owner.WriteNote(owned, "Food/Soup.md", "carrots and leeks")
	w.owner.WriteNote(owned, "Food/Bread.md", "flour")
	w.syncOwner(t)
	if err := w.reader.Sync(t.Context(), received.VaultID); err != nil {
		t.Fatal(err)
	}

	wantNote(t, w.reader, received.VaultID, "Soup.md", "carrots and leeks")
	wantNote(t, w.reader, received.VaultID, "Bread.md", "flour")
}

func TestPullShare_MakesNoWriteCallAndNeverReadsTheRevisionOrBlobServicesOfTheOwner(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	received := w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))
	procedures := []string{
		syncv1connect.RevisionServiceCommitRevisionsProcedure, syncv1connect.RevisionServiceGetChangesProcedure,
		syncv1connect.BlobServiceReserveBlobsProcedure, syncv1connect.BlobServiceGetBlobUrlsProcedure,
		syncv1connect.CatalogServiceAppendCatalogEntryProcedure, syncv1connect.ShareServiceAppendShareLogEntryProcedure,
	}
	before := make([]int, len(procedures))
	w.owner.WriteNote(owned, "Food/Bread.md", "flour")
	w.syncOwner(t)
	for i, procedure := range procedures {
		before[i] = w.server.Fake.CountCalls(procedure)
	}

	if err := w.reader.Sync(t.Context(), received.VaultID); err != nil {
		t.Fatal(err)
	}

	for i, procedure := range procedures {
		if got := w.server.Fake.CountCalls(procedure); got != before[i] {
			t.Errorf("%s ran %d times for a pass of the reader", procedure, got-before[i])
		}
	}
	if w.server.Fake.CountCalls(syncv1connect.ShareServiceGetShareChangesProcedure) == 0 {
		t.Error("the pass of the reader did not read the share feed")
	}
}

func TestPullShare_KeepsAFileTheReaderChangedAndCountsIt(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	received := w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))
	w.reader.WriteNote(received.VaultID, "Soup.md", "my own soup")
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots and leeks")
	w.syncOwner(t)

	if err := w.reader.Sync(t.Context(), received.VaultID); err != nil {
		t.Fatal(err)
	}

	wantNote(t, w.reader, received.VaultID, "Soup.md", "my own soup")
	listed, err := w.reader.ListReceived(t.Context())
	if err != nil || len(listed) != 1 || listed[0].ConflictCount != 1 {
		t.Errorf("received = %+v, error = %v, want one conflict counted", listed, err)
	}
}

func TestPullShare_RemovesAFileTheOwnerMovedOutOfTheFolderUnlessTheReaderChangedIt(t *testing.T) {
	t.Skip("waits for the owner side to push moves and removals (#2569, #2570)")
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.owner.WriteNote(owned, "Food/Stew.md", "beef")
	w.syncOwner(t)
	received := w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))
	w.reader.WriteNote(received.VaultID, "Stew.md", "my own stew")

	w.owner.MoveNote(t, owned, "Food/Soup.md", "Plans")
	w.owner.MoveNote(t, owned, "Food/Stew.md", "Plans")
	w.syncOwner(t)
	if err := w.reader.Sync(t.Context(), received.VaultID); err != nil {
		t.Fatal(err)
	}

	wantNoNote(t, w.reader, received.VaultID, "Soup.md")
	wantNote(t, w.reader, received.VaultID, "Stew.md", "my own stew")
}

func TestPullShare_RemovesAFileTheOwnerDeleted(t *testing.T) {
	t.Skip("waits for the owner side to push removals (#2570)")
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	received := w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))

	w.owner.Vaults.With(owned).Remove("Food/Soup.md")
	w.syncOwner(t)
	if err := w.reader.Sync(t.Context(), received.VaultID); err != nil {
		t.Fatal(err)
	}

	wantNoNote(t, w.reader, received.VaultID, "Soup.md")
}

func TestPullShare_AReaderRemovedByTheOwnerKeepsItsFilesGetsNothingNewAndShowsTheEnd(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	invitation := w.shareWith(t, w.reader, "Anna")
	received := w.receive(t, w.reader, invitation)
	if err := w.owner.RemoveReader(t.Context(), owned, folder, invitation.ReaderFingerprint); err != nil {
		t.Fatal(err)
	}
	w.owner.WriteNote(owned, "Food/Bread.md", "flour")
	w.syncOwner(t)

	err := w.reader.Sync(t.Context(), received.VaultID)

	if !errors.Is(err, share.ErrEnded) {
		t.Errorf("error = %v, want ErrEnded", err)
	}
	wantNote(t, w.reader, received.VaultID, "Soup.md", "carrots")
	wantNoNote(t, w.reader, received.VaultID, "Bread.md")
	listed, listErr := w.reader.ListReceived(t.Context())
	if listErr != nil || len(listed) != 1 || listed[0].State != "ended" {
		t.Errorf("received = %+v, error = %v, want the share ended", listed, listErr)
	}
	before := w.server.Fake.CountCalls(syncv1connect.ShareServiceGetShareChangesProcedure)
	if err := w.reader.Sync(t.Context(), received.VaultID); err != nil && !errors.Is(err, share.ErrEnded) {
		t.Fatal(err)
	}
	if got := w.server.Fake.CountCalls(syncv1connect.ShareServiceGetShareChangesProcedure); got != before {
		t.Errorf("an ended share asked the server for changes %d more times, want none", got-before)
	}
}

func TestPullShare_TheReaderThatRemainsReadsTheNewEpochAndALaterReaderReadsEveryEpoch(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	removed := w.shareWith(t, w.reader, "Anna")
	staying := synctest.NewReaderComputer(t, w.server, w.clock, "Staying")
	stayingInvitation := w.shareWith(t, staying, "Bo")
	stayingVault := w.receive(t, staying, stayingInvitation)
	if err := w.owner.RemoveReader(t.Context(), owned, folder, removed.ReaderFingerprint); err != nil {
		t.Fatal(err)
	}
	w.owner.WriteNote(owned, "Food/Bread.md", "flour")
	w.syncOwner(t)
	late := synctest.NewReaderComputer(t, w.server, w.clock, "Late")
	lateVault := w.receive(t, late, w.shareWith(t, late, "Cy"))

	if err := staying.Sync(t.Context(), stayingVault.VaultID); err != nil {
		t.Fatal(err)
	}

	wantNote(t, staying, stayingVault.VaultID, "Bread.md", "flour")
	wantNote(t, staying, stayingVault.VaultID, "Soup.md", "carrots")
	wantNote(t, late, lateVault.VaultID, "Bread.md", "flour")
	wantNote(t, late, lateVault.VaultID, "Soup.md", "carrots")
}

func TestPullShare_AReaderWithoutASubscriptionReadsTheFolderAndMakesNoWriteCall(t *testing.T) {
	w := newWorldOn(t, fakeserver.Config{NeedsSubscription: true})
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	received := w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))
	wantNote(t, w.reader, received.VaultID, "Soup.md", "carrots")
	w.owner.WriteNote(owned, "Food/Bread.md", "flour")
	w.syncOwner(t)
	commits := w.server.Fake.CountCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure)
	reserves := w.server.Fake.CountCalls(syncv1connect.BlobServiceReserveBlobsProcedure)

	if err := w.reader.Sync(t.Context(), received.VaultID); err != nil {
		t.Fatal(err)
	}

	wantNote(t, w.reader, received.VaultID, "Bread.md", "flour")
	if got := w.server.Fake.CountCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure); got != commits {
		t.Errorf("the reader committed %d revisions, want none", got-commits)
	}
	if got := w.server.Fake.CountCalls(syncv1connect.BlobServiceReserveBlobsProcedure); got != reserves {
		t.Errorf("the reader reserved %d blobs, want none", got-reserves)
	}
	if w.reader.HasItemKey("subscription-needed") {
		t.Error("the health host of the reader asks for a subscription for a share it reads for free")
	}
}
