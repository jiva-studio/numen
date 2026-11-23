package share_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
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

func newWorld(t *testing.T) *world {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	owner := synctest.NewFirstComputer(t, server, clock, string(owned), synctest.NewWords(t))
	owner.Connect(t, owned)
	reader := synctest.NewReaderComputer(t, server, clock, "Reader")
	return &world{server: server, clock: clock, owner: owner, reader: reader}
}

func (w *world) syncOwner(t *testing.T) {
	t.Helper()
	if err := w.owner.Sync(t.Context(), owned); err != nil {
		t.Fatalf("pass of the owner: %v", err)
	}
}

func (w *world) heads(t *testing.T) map[string]*syncv1.StoredRevision {
	t.Helper()
	heads := map[string]*syncv1.StoredRevision{}
	for _, head := range w.owner.ReadChanges(t, w.owner.GetRemoteID(t, owned)) {
		heads[string(head.GetRevision().GetFileId())] = head
	}
	return heads
}

func (w *world) addReader(t *testing.T, reader *synctest.Computer, label string) synctest.ShareInvitation {
	t.Helper()
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

func (w *world) shareFolder(t *testing.T) {
	t.Helper()
	if err := w.owner.CreateShare(t.Context(), owned, folder); err != nil {
		t.Fatal(err)
	}
}

func TestCreateShare_RewrapsOnlyFilesInFolderAndReusesBlobs(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.owner.WriteNote(owned, "Food/Dinner/Stew.md", "beef")
	w.owner.WriteNote(owned, "Plans/Trip.md", "Lisbon")
	w.syncOwner(t)
	before := w.heads(t)

	w.shareFolder(t)
	w.syncOwner(t)

	shared, other := 0, 0
	for id, head := range w.heads(t) {
		revision := head.GetRevision()
		if len(revision.GetShareRootId()) == 0 {
			other++
			if !bytes.Equal(head.GetRevId(), before[id].GetRevId()) {
				t.Error("a file outside the folder got a new revision")
			}
			continue
		}
		shared++
		if revision.GetEpoch() != 0 || !bytes.Equal(revision.GetBlobId(), before[id].GetRevision().GetBlobId()) ||
			!bytes.Equal(revision.GetBlobHash(), before[id].GetRevision().GetBlobHash()) {
			t.Errorf("share revision = %v, want epoch 0 and the blob of the vault revision", revision)
		}
		if !bytes.Equal(revision.GetParentRev(), before[id].GetRevId()) {
			t.Error("the share revision does not have the vault revision as its parent")
		}
	}
	if shared != 2 || other != 1 {
		t.Errorf("%d share revisions and %d others, want 2 and 1", shared, other)
	}
}

func TestCreateShare_PreservesOwnerFilesAndNextPassWritesNothing(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	w.shareFolder(t)
	w.syncOwner(t)
	heads := w.heads(t)

	w.syncOwner(t)

	if got, ok := w.owner.ReadNote(owned, "Food/Soup.md"); !ok || got != "carrots" {
		t.Errorf("note of the owner = %q (%v), want carrots", got, ok)
	}
	for id, head := range w.heads(t) {
		if !bytes.Equal(head.GetRevId(), heads[id].GetRevId()) {
			t.Error("a pass over a shared vault made a new revision with nothing changed")
		}
	}
}

func TestCreateShare_HidesThePathAboveTheFolderFromTheServer(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	w.shareFolder(t)
	w.syncOwner(t)

	for _, head := range w.heads(t) {
		if bytes.Contains(head.GetRevision().GetEncMeta(), []byte("Soup")) || bytes.Contains(head.GetRevision().GetEncMeta(), []byte("Food")) {
			t.Error("the enc_meta of a share revision shows a path")
		}
	}
	if bytes.Contains(w.server.GetStoredBlobs(t), []byte("carrots")) {
		t.Error("the server holds a plain text")
	}
}

func TestCreateShare_RefusesTheWholeVaultNestingAReceivedVaultAndAVaultThatIsNotConnected(t *testing.T) {
	w := newWorld(t)
	w.shareFolder(t)
	w.owner.AddVault(t, "Offline")

	for name, c := range map[string]struct {
		vault  domain.VaultID
		folder string
		want   error
	}{
		"the vault root":     {vault: owned, folder: "", want: share.ErrWholeVault},
		"the same folder":    {vault: owned, folder: folder, want: share.ErrInsideShare},
		"inside the folder":  {vault: owned, folder: "Food/Dinner", want: share.ErrInsideShare},
		"a vault not synced": {vault: "Offline", folder: "Food", want: share.ErrNotConnected},
	} {
		if err := w.owner.CreateShare(t.Context(), c.vault, c.folder); !errors.Is(err, c.want) {
			t.Errorf("%s: error = %v, want %v", name, err, c.want)
		}
	}
	if err := w.owner.CreateShare(t.Context(), owned, "Life"); err != nil {
		t.Fatal(err)
	}
	if err := w.owner.CreateShare(t.Context(), owned, ""); !errors.Is(err, share.ErrWholeVault) {
		t.Errorf("the root with two shares: error = %v, want ErrWholeVault", err)
	}
}

func TestCreateShare_RejectsFolderContainingSharedFolderOnSecondComputer(t *testing.T) {
	words := synctest.NewWords(t)
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	first := synctest.NewFirstComputer(t, server, clock, string(owned), words)
	first.Connect(t, owned)
	second := synctest.NewSecondComputer(t, server, clock, "Copy", words)
	second.ConnectExisting(t, "Copy", first.Link(t, owned).RemoteVaultID)
	if err := first.CreateShare(t.Context(), owned, "Life/Food"); err != nil {
		t.Fatal(err)
	}
	if err := second.Sync(t.Context(), "Copy"); err != nil {
		t.Fatal(err)
	}

	if err := second.CreateShare(t.Context(), "Copy", "Life"); !errors.Is(err, share.ErrHoldsShare) {
		t.Errorf("error = %v, want ErrHoldsShare", err)
	}
}

func TestCreateShare_RefusesAReceivedVault(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	w.shareFolder(t)
	received, err := w.reader.AcceptInvitation(t.Context(), w.addReader(t, w.reader, "Reader").Text)
	if err != nil {
		t.Fatal(err)
	}

	if err := w.reader.CreateShare(t.Context(), received.VaultID, "Dinner"); !errors.Is(err, share.ErrReceivedVault) {
		t.Errorf("error = %v, want ErrReceivedVault", err)
	}
	if err := w.reader.ConnectVault(t.Context(), received.VaultID); !errors.Is(err, share.ErrReceivedNotSynced) {
		t.Errorf("connect: error = %v, want ErrReceivedNotSynced", err)
	}
}

func TestFileAddedToFolderAfterShareIsCommittedAsShareRevisionAtCurrentEpoch(t *testing.T) {
	w := newWorld(t)
	w.shareFolder(t)
	w.owner.WriteNote(owned, "Food/Bread.md", "flour")
	w.owner.WriteNote(owned, "Plans/Trip.md", "Lisbon")

	w.syncOwner(t)

	var shared, plain int
	for _, head := range w.heads(t) {
		if len(head.GetRevision().GetShareRootId()) == 16 {
			shared++
		} else {
			plain++
		}
	}
	if shared != 1 || plain != 1 {
		t.Errorf("%d share revisions and %d plain ones, want one of each", shared, plain)
	}
}

func TestAddReader_ReturnsAnInvitationForTheFingerprintOfTheReader(t *testing.T) {
	w := newWorld(t)
	w.shareFolder(t)
	code, err := w.reader.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	invitation, err := w.owner.AddReader(t.Context(), owned, folder, code.Code, "Anna", false)
	if err != nil {
		t.Fatal(err)
	}

	if invitation.ReaderFingerprint != code.Fingerprint || invitation.Text == "" {
		t.Errorf("invitation = %+v, want the text and the fingerprint %s", invitation, code.Fingerprint)
	}
	state, err := w.owner.GetShare(t.Context(), owned, folder)
	if err != nil || state.State != "active" || len(state.Readers) != 1 || state.Readers[0].Fingerprint != code.Fingerprint || state.Readers[0].Label != "Anna" {
		t.Errorf("share = %+v, error = %v, want one reader Anna with the fingerprint of the code", state, err)
	}
}

func TestAddReader_RefusesTheOwnIdentityAnActiveReaderAndAReaderNotSignedIn(t *testing.T) {
	w := newWorld(t)
	w.shareFolder(t)
	own, err := w.owner.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	w.addReader(t, w.reader, "Anna")
	again, err := w.reader.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	absent, err := synctest.NewUnsignedComputer(t, w.server, w.clock, "Absent").GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	for name, c := range map[string]struct {
		code string
		want error
	}{
		"the own identity":      {code: own.Code, want: share.ErrOwnIdentity},
		"an active reader":      {code: again.Code, want: share.ErrAlreadyReader},
		"refused by the server": {code: absent.Code, want: share.ErrReaderNotAdded},
		"not a code":            {code: "numen-reader-1.AAAA", want: share.ErrBadCode},
	} {
		if _, err := w.owner.AddReader(t.Context(), owned, folder, c.code, "Bo", false); !errors.Is(err, c.want) {
			t.Errorf("%s: error = %v, want %v", name, err, c.want)
		}
	}
}

func TestAddReader_WarnsOfALabelWithAnotherRootUntilTheChangeIsConfirmed(t *testing.T) {
	w := newWorld(t)
	w.shareFolder(t)
	w.addReader(t, w.reader, "Anna")
	second := synctest.NewReaderComputer(t, w.server, w.clock, "Other")
	code, err := second.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := w.owner.AddReader(t.Context(), owned, folder, code.Code, "Anna", false); !errors.Is(err, share.ErrChangedIdentity) {
		t.Fatalf("error = %v, want ErrChangedIdentity", err)
	}
	if _, err := w.owner.AddReader(t.Context(), owned, folder, code.Code, "Anna", true); err != nil {
		t.Errorf("confirmed: error = %v", err)
	}
}

func TestAddReader_WaitsForTheRewrapToEnd(t *testing.T) {
	w := newWorld(t)
	code, err := w.reader.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.owner.AddReader(t.Context(), owned, folder, code.Code, "Anna", false)

	if !errors.Is(err, share.ErrNotReady) {
		t.Errorf("a reader added to a folder that is not shared: error = %v, want ErrNotReady", err)
	}
}

func TestRemoveReader_RotatesTheEpochAndTheNextRevisionOfTheOwnerCarriesIt(t *testing.T) {
	w := newWorld(t)
	w.shareFolder(t)
	first := w.addReader(t, w.reader, "Anna")
	other := synctest.NewReaderComputer(t, w.server, w.clock, "Other")
	w.addReader(t, other, "Bo")

	if err := w.owner.RemoveReader(t.Context(), owned, folder, first.ReaderFingerprint); err != nil {
		t.Fatal(err)
	}
	w.owner.WriteNote(owned, "Food/Bread.md", "flour")
	w.syncOwner(t)

	state, err := w.owner.GetShare(t.Context(), owned, folder)
	if err != nil || len(state.Readers) != 2 {
		t.Fatalf("share = %+v, error = %v, want both readers listed", state, err)
	}
	var epochs []uint32
	for _, head := range w.heads(t) {
		if len(head.GetRevision().GetShareRootId()) == 16 {
			epochs = append(epochs, head.GetRevision().GetEpoch())
		}
	}
	if len(epochs) != 1 || epochs[0] != 1 {
		t.Errorf("epochs of the share revisions = %v, want [1]", epochs)
	}
	if got := w.server.Fake.CountCalls(syncv1connect.ShareServiceAppendShareLogEntryProcedure); got < 4 {
		t.Errorf("%d log entries appended, want CREATE, two ADD_READER and REMOVE_READER", got)
	}
}

func TestStopShare_MakesLaterRevisionsOfTheFolderPlainVaultRevisions(t *testing.T) {
	w := newWorld(t)
	w.shareFolder(t)
	w.addReader(t, w.reader, "Anna")

	if err := w.owner.StopShare(t.Context(), owned, folder); err != nil {
		t.Fatal(err)
	}
	w.owner.WriteNote(owned, "Food/Bread.md", "flour")
	w.syncOwner(t)

	for _, head := range w.heads(t) {
		if len(head.GetRevision().GetShareRootId()) != 0 {
			t.Error("a revision after the share stopped names the share")
		}
	}
	state, err := w.owner.GetShare(t.Context(), owned, folder)
	if err != nil || state.State != "notShared" {
		t.Errorf("share = %+v, error = %v, want notShared", state, err)
	}
}

func TestCreateShare_ResumesAnInterruptedRewrapAtItsCursorAndAddsNoReaderBeforeItEnds(t *testing.T) {
	w := newWorld(t)
	for _, name := range []string{"A", "B", "C"} {
		w.owner.WriteNote(owned, "Food/"+name+".md", name)
	}
	w.syncOwner(t)
	w.server.LoseNextResponse("/CommitRevisions")

	if err := w.owner.CreateShare(t.Context(), owned, folder); err == nil {
		t.Fatal("a share whose rewrap lost a response reported success")
	}
	code, err := w.reader.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.owner.AddReader(t.Context(), owned, folder, code.Code, "Anna", false); !errors.Is(err, share.ErrNotReady) {
		t.Errorf("a reader added during the rewrap: error = %v, want ErrNotReady", err)
	}
	w.syncOwner(t)

	shared := 0
	for _, head := range w.heads(t) {
		if len(head.GetRevision().GetShareRootId()) == 16 {
			shared++
		}
	}
	if shared != 3 {
		t.Errorf("%d share revisions after the pass resumed the rewrap, want 3", shared)
	}
}
