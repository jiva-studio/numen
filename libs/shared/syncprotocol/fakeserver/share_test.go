package fakeserver_test

import (
	"bytes"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

type sharing struct {
	owner   *env
	reader  *env
	server  *fakeserver.Server
	owned   *syncclient.ShareClient
	read    *syncclient.ShareClient
	vault   []byte
	share   []byte
	version uint64
	head    [32]byte
	epoch   uint32
}

func newShareClientOf(e *env) *syncclient.ShareClient {
	return syncclient.NewShareClient(e.address, http.DefaultClient, func() string { return e.token })
}

func newSharingHarness(tb testing.TB) *sharing {
	tb.Helper()
	server, address := newServer(tb)
	account := newOwnedAccount(tb)
	owner := newEnvOn(tb, address, &account, account.addDevice(tb, "First"))
	reader := newSecondAccountOn(tb, address)
	s := &sharing{owner: owner, reader: reader, server: server, owned: newShareClientOf(owner), read: newShareClientOf(reader), vault: owner.newVault(), share: newID(16)}
	return s
}

func (s *sharing) readerRoot() []byte { return s.reader.account.chain.Root().Sign }

func (s *sharing) ownerRoot() []byte { return s.owner.account.chain.Root().Sign }

func newSealFor(root []byte) *syncv1.ShareSeal {
	return &syncv1.ShareSeal{ReaderRootSignKey: root, Sealed: bytes.Repeat([]byte{7}, 84)}
}

func (s *sharing) newEntry(kind syncv1.ShareLogKind, epoch uint32, subject []byte, seals ...*syncv1.ShareSeal) *syncv1.ShareLogEntry {
	entry := &syncv1.ShareLogEntry{
		ShareRootId: s.share, Version: s.version + 1, Kind: kind, Epoch: epoch, SubjectRootSignKey: subject,
		EncOwnerBody: []byte("owner body"), EncShareBody: []byte("share body"), Seals: seals,
		AuthorDeviceSignKey: s.owner.device.Public().Sign,
	}
	if s.version > 0 {
		entry.PrevHash = bytes.Clone(s.head[:])
	}
	if kind == syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE {
		entry.VaultId = s.vault
	}
	if epoch > 0 && (kind == syncv1.ShareLogKind_SHARE_LOG_KIND_REMOVE_READER || kind == syncv1.ShareLogKind_SHARE_LOG_KIND_ROTATE) {
		entry.PrevKeySeal = []byte("previous key")
	}
	signature := s.owner.device.SignForPlugin(syncprotocol.SignInPluginID, syncprotocol.GetShareLogEntryMessage(entry))
	entry.Signature = signature[:]
	return entry
}

func (s *sharing) tryAppend(entry *syncv1.ShareLogEntry) (*syncv1.AppendShareLogEntryResponse, error) {
	response, err := s.owned.AppendShareLogEntry(s.owner.tb.Context(), entry)
	if err == nil && response.GetCommitted() {
		s.version, s.head, s.epoch = entry.GetVersion(), syncprotocol.GetShareLogEntryID(entry), entry.GetEpoch()
	}
	return response, err
}

func (s *sharing) mustAppend(entry *syncv1.ShareLogEntry) {
	s.owner.tb.Helper()
	response, err := s.tryAppend(entry)
	if err != nil || !response.GetCommitted() {
		s.owner.tb.Fatalf("entry %d: committed = %v, error = %v", entry.GetVersion(), response.GetCommitted(), err)
	}
}

func (s *sharing) create() {
	s.owner.tb.Helper()
	s.mustAppend(s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE, 0, nil))
}

func (s *sharing) addReader() {
	s.owner.tb.Helper()
	s.mustAppend(s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER, s.epoch, s.readerRoot(), newSealFor(s.readerRoot())))
}

func (s *sharing) removeReader(remaining ...*syncv1.ShareSeal) {
	s.owner.tb.Helper()
	s.mustAppend(s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_REMOVE_READER, s.epoch+1, s.readerRoot(), remaining...))
}

func (s *sharing) commitShared(path string, epoch uint32, body []byte) (*syncv1.Revision, *syncv1.CommitRevisionsResponse) {
	s.owner.tb.Helper()
	revision := s.owner.newFile(s.vault, newID(16), nil, path, body)
	revision.ShareRootId, revision.Epoch = s.share, epoch
	s.owner.signRevision(revision)
	return revision, s.owner.commit(s.vault, revision)
}

func TestShareService_CreateMakesAShareThatTheOwnerListsAtEpochZero(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)

	s.create()

	shares, err := s.owned.ListShares(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 1 || !bytes.Equal(shares[0].GetShareRootId(), s.share) || !bytes.Equal(shares[0].GetVaultId(), s.vault) ||
		shares[0].GetHeadVersion() != 1 || shares[0].GetEpoch() != 0 || shares[0].GetIsClosed() {
		t.Errorf("shares = %v, want the open share of the vault at version 1 and epoch 0", shares)
	}
	if listed, err := s.read.ListShares(t.Context()); err != nil || len(listed) != 0 {
		t.Errorf("a reader lists %v, %v, want no share", listed, err)
	}
}

func TestShareService_CreateRefusesAnExistingIdAndAVaultOfAnotherAccount(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	s.create()
	again := s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE, 0, nil)
	again.Version, again.PrevHash = 1, nil
	signature := s.owner.device.SignForPlugin(syncprotocol.SignInPluginID, syncprotocol.GetShareLogEntryMessage(again))
	again.Signature = signature[:]
	if _, err := s.owned.AppendShareLogEntry(t.Context(), again); !errors.Is(err, syncprotocol.ErrShareExists) {
		t.Errorf("a second CREATE error = %v, want ErrShareExists", err)
	}

	foreign := newSharingHarness(t)
	foreign.vault = s.vault
	if _, err := foreign.tryAppend(foreign.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE, 0, nil)); !errors.Is(err, syncprotocol.ErrNoVault) {
		t.Errorf("CREATE for a vault of another account: error = %v, want ErrNoVault", err)
	}
}

func TestShareService_AnswersAStaleVersionWithTheHeadAndStoresNothing(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	s.create()
	stale := s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_UPDATE, 0, nil)
	stale.EncShareBody = []byte("share body of another computer")
	signature := s.owner.device.SignForPlugin(syncprotocol.SignInPluginID, syncprotocol.GetShareLogEntryMessage(stale))
	stale.Signature = signature[:]
	if _, err := s.tryAppend(s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_UPDATE, 0, nil)); err != nil {
		t.Fatal(err)
	}

	response, err := s.owned.AppendShareLogEntry(t.Context(), stale)
	if err != nil {
		t.Fatal(err)
	}

	if response.GetCommitted() || response.GetHead().GetVersion() != 2 {
		t.Errorf("response = %v, want not committed with the head at version 2", response)
	}
}

func TestShareService_RefusesAnEntryNotSignedByADeviceOfTheOwner(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	entry := s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE, 0, nil)
	entry.Signature[0] ^= 1

	if _, err := s.owned.AppendShareLogEntry(t.Context(), entry); err == nil {
		t.Error("an entry with a changed signature was accepted")
	}
}

func TestShareService_AddReaderNeedsAnAccountOfTheRootAndExactlyOneSealForIt(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	s.create()
	stranger := bytes.Repeat([]byte{9}, 32)

	_, unknown := s.tryAppend(s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER, 0, stranger, newSealFor(stranger)))
	_, wrongSeal := s.tryAppend(s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER, 0, s.readerRoot(), newSealFor(stranger)))
	_, own := s.tryAppend(s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER, 0, s.ownerRoot(), newSealFor(s.ownerRoot())))

	if unknown == nil || wrongSeal == nil || own == nil {
		t.Fatalf("a root without an account, a seal for another key or the owner as reader was accepted: %v, %v, %v", unknown, wrongSeal, own)
	}
	if !errors.Is(unknown, syncprotocol.ErrReaderRefused) || !errors.Is(own, syncprotocol.ErrReaderRefused) || connect.CodeOf(unknown) != connect.CodeOf(own) {
		t.Errorf("a root without an account answers %v and the owner as reader %v, want one refusal that says nothing of the accounts", connect.CodeOf(unknown), connect.CodeOf(own))
	}
}

func TestShareService_AMemberReadsTheLogWithTheChainOfTheOwnerAndAStrangerGetsNoShare(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	s.create()
	s.addReader()

	entries, chain, err := s.read.GetShareLog(t.Context(), s.share, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || !bytes.Equal(chain, s.owner.account.chain.Bytes()) {
		t.Errorf("entries = %d and chain of %d bytes, want 2 entries and the chain of the owner", len(entries), len(chain))
	}
	other := newSecondAccountOn(t, s.owner.address)
	if _, _, err := newShareClientOf(other).GetShareLog(t.Context(), s.share, 0); !errors.Is(err, syncprotocol.ErrNoShare) {
		t.Errorf("a stranger: error = %v, want ErrNoShare", err)
	}
	if _, _, err := s.read.GetShareLog(t.Context(), newID(16), 0); !errors.Is(err, syncprotocol.ErrNoShare) {
		t.Errorf("an unknown share: error = %v, want ErrNoShare", err)
	}
}

func TestShareService_ReaderReadsAShareWithoutASubscriptionAndMakesNoWriteCall(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	s.create()
	s.addReader()

	if _, _, err := s.read.GetShareLog(t.Context(), s.share, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.read.GetShareChanges(t.Context(), s.share, 0, 0); err != nil {
		t.Fatal(err)
	}

	for _, procedure := range []string{"/numen.sync.v1.RevisionService/CommitRevisions", "/numen.sync.v1.BlobService/ReserveBlobs"} {
		if got := s.server.CountCalls(procedure); got != 0 {
			t.Errorf("%s ran %d times for a reader", procedure, got)
		}
	}
	if _, err := s.read.AppendShareLogEntry(t.Context(), s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_UPDATE, 0, nil)); !errors.Is(err, syncprotocol.ErrNoShare) {
		t.Errorf("a reader appends: error = %v, want ErrNoShare", err)
	}
}

func TestShareService_RemoveReaderNeedsTheNextEpochAndASealForEachRemainingReader(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	s.create()
	s.addReader()
	second := newSecondAccountOn(t, s.owner.address)
	secondRoot := second.account.chain.Root().Sign
	s.mustAppend(s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER, 0, secondRoot, newSealFor(secondRoot)))

	_, sameEpoch := s.tryAppend(s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_REMOVE_READER, 0, s.readerRoot(), newSealFor(secondRoot)))
	_, noSeal := s.tryAppend(s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_REMOVE_READER, 1, s.readerRoot()))
	_, sealOfRemoved := s.tryAppend(s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_REMOVE_READER, 1, s.readerRoot(), newSealFor(secondRoot), newSealFor(s.readerRoot())))

	if sameEpoch == nil || noSeal == nil || sealOfRemoved == nil {
		t.Fatalf("a removal with the same epoch (%v), no seal (%v) or a seal for the removed reader (%v) was accepted", sameEpoch, noSeal, sealOfRemoved)
	}
	s.removeReader(newSealFor(secondRoot))
	shares, err := s.owned.ListShares(t.Context())
	if err != nil || len(shares) != 1 || shares[0].GetEpoch() != 1 || shares[0].GetHeadVersion() != 4 {
		t.Errorf("shares = %v, %v, want epoch 1 at version 4", shares, err)
	}
}

func TestShareService_ARemovedReaderGetsTheLogUpToItsRemovalAndShareEndedAfter(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	s.create()
	s.addReader()
	s.removeReader()
	s.mustAppend(s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_UPDATE, 1, nil))

	entries, _, err := s.read.GetShareLog(t.Context(), s.share, 0)
	if err != nil || len(entries) != 3 {
		t.Fatalf("entries = %d, error = %v, want the log up to the removal", len(entries), err)
	}
	_, changesErr := s.read.GetShareChanges(t.Context(), s.share, 0, 0)
	_, urlsErr := s.read.GetShareBlobUrls(t.Context(), s.share, [][]byte{newID(16)})
	for name, err := range map[string]error{"changes": changesErr, "urls": urlsErr} {
		var ended syncprotocol.ShareEndedError
		if !errors.As(err, &ended) || ended.Version != 3 {
			t.Errorf("%s: error = %v, want ShareEndedError at version 3", name, err)
		}
	}
}

func TestShareService_CloseEndsEveryReaderAndTheLogAcceptsNothingAfterIt(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	s.create()
	s.addReader()
	s.mustAppend(s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_CLOSE, 0, nil))

	_, err := s.read.GetShareChanges(t.Context(), s.share, 0, 0)
	var ended syncprotocol.ShareEndedError
	if !errors.As(err, &ended) || ended.Version != 3 {
		t.Errorf("a reader of a closed share: error = %v, want ShareEndedError at version 3", err)
	}
	if _, err := s.tryAppend(s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_UPDATE, 0, nil)); !errors.Is(err, syncprotocol.ErrNoShare) {
		t.Errorf("an append after CLOSE: error = %v, want ErrNoShare", err)
	}
	shares, err := s.owned.ListShares(t.Context())
	if err != nil || len(shares) != 1 || !shares[0].GetIsClosed() {
		t.Errorf("shares = %v, %v, want the share closed", shares, err)
	}
}

func TestShareService_ReaderFeedHasOnlyRevisionsOfTheShareAndNoVaultSeq(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	s.create()
	s.addReader()
	shared, _ := s.commitShared("Food/milk.md", 0, []byte("milk"))
	s.owner.commit(s.vault, s.owner.newFile(s.vault, newID(16), nil, "Test/secret.md", []byte("secret")))

	page, err := s.read.GetShareChanges(t.Context(), s.share, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(page.GetChanges()) != 1 || page.GetChanges()[0].GetLeft() ||
		!bytes.Equal(page.GetChanges()[0].GetChange().GetRevision().GetFileId(), shared.GetFileId()) {
		t.Fatalf("changes = %v, want the one file of the share", page.GetChanges())
	}
	hash := hashOf(shared.GetEncMeta())
	if !bytes.Equal(page.GetChanges()[0].GetEncMetaHash(), hash) {
		t.Errorf("enc_meta_hash = %x, want the SHA-256 of enc_meta", page.GetChanges()[0].GetEncMetaHash())
	}
	if page.GetNextSeq() != page.GetChanges()[0].GetChange().GetSeq() {
		t.Errorf("next_seq = %d, want the seq of the page", page.GetNextSeq())
	}
}

func TestShareService_AFileThatLeftIsServedAsItsSignedHeadWithoutMetaAndWithItsHash(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	s.create()
	s.addReader()
	shared, _ := s.commitShared("Food/milk.md", 0, []byte("milk"))
	storedID := syncprotocol.GetRevisionID(shared)
	moved := &syncv1.Revision{
		VaultId: s.vault, FileId: shared.GetFileId(), ParentRev: storedID[:], BlobId: shared.GetBlobId(), BlobHash: shared.GetBlobHash(), BlobSize: shared.GetBlobSize(),
		EncMeta: []byte("meta under the vault key"), PathHash: hashOf([]byte("Moved/milk.md")), AuthorDeviceSignKey: s.owner.device.Public().Sign,
	}
	s.owner.signRevision(moved)
	s.owner.commit(s.vault, moved)

	page, err := s.read.GetShareChanges(t.Context(), s.share, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(page.GetChanges()) != 1 || !page.GetChanges()[0].GetLeft() {
		t.Fatalf("changes = %v, want the file as left", page.GetChanges())
	}
	change := page.GetChanges()[0]
	if len(change.GetChange().GetRevision().GetEncMeta()) != 0 {
		t.Error("a file that left was served with its enc_meta")
	}
	var metaHash [32]byte
	copy(metaHash[:], change.GetEncMetaHash())
	if !bytes.Equal(metaHash[:], hashOf(moved.GetEncMeta())) {
		t.Errorf("enc_meta_hash = %x, want the SHA-256 of the meta the owner signed", metaHash)
	}
	if got := syncprotocol.GetRevisionIDWithMetaHash(change.GetChange().GetRevision(), metaHash); !bytes.Equal(got[:], change.GetChange().GetRevId()) {
		t.Error("the id computed with the meta hash differs from rev_id")
	}
	if len(change.GetChange().GetRevision().GetShareRootId()) != 0 {
		t.Error("the served head still names the share")
	}
}

func TestShareService_ABlobUrlIsGivenForTheHeadOfAShareFileOnly(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	s.create()
	s.addReader()
	shared, _ := s.commitShared("Food/milk.md", 0, []byte("milk"))
	outside := s.owner.newFile(s.vault, newID(16), nil, "Test/secret.md", []byte("secret"))
	s.owner.commit(s.vault, outside)

	downloads, err := s.read.GetShareBlobUrls(t.Context(), s.share, [][]byte{shared.GetBlobId(), outside.GetBlobId(), newID(16)})
	if err != nil {
		t.Fatal(err)
	}

	if len(downloads) != 3 || !downloads[0].GetFound() || downloads[1].GetFound() || downloads[2].GetFound() {
		t.Errorf("downloads = %v, want found for the share file only", downloads)
	}
	if downloads[1].GetUrl() != "" || downloads[2].GetUrl() != "" {
		t.Error("a url was given for a blob outside the share")
	}
}

func TestCommitRevisions_RefusesAShareIdThatIsNotAnOpenShareOfTheVault(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	revision := s.owner.newFile(s.vault, newID(16), nil, "Food/milk.md", []byte("milk"))
	revision.ShareRootId = newID(16)
	s.owner.signRevision(revision)

	_, err := s.owner.revision.CommitRevisions(t.Context(), s.vault, []*syncv1.Revision{revision})

	if !errors.Is(err, syncprotocol.ErrInvalidRevision) {
		t.Errorf("an unknown share id: error = %v, want ErrInvalidRevision", err)
	}
	s.create()
	s.mustAppend(s.newEntry(syncv1.ShareLogKind_SHARE_LOG_KIND_CLOSE, 0, nil))
	closed := s.owner.newFile(s.vault, newID(16), nil, "Food/bread.md", []byte("bread"))
	closed.ShareRootId = s.share
	s.owner.signRevision(closed)
	if _, err := s.owner.revision.CommitRevisions(t.Context(), s.vault, []*syncv1.Revision{closed}); !errors.Is(err, syncprotocol.ErrInvalidRevision) {
		t.Errorf("a closed share id: error = %v, want ErrInvalidRevision", err)
	}
}

func TestCommitRevisions_AnswersShareEpochForARevisionOfAnotherEpochAndStoresNothing(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	s.create()
	s.addReader()
	s.removeReader()

	_, response := s.commitShared("Food/milk.md", 0, []byte("milk"))

	if response.GetCommitted() || len(response.GetConflicts()) != 1 {
		t.Fatalf("response = %v, want one conflict and no commit", response)
	}
	conflict := response.GetConflicts()[0]
	if conflict.GetKind() != syncv1.ConflictKind_CONFLICT_KIND_SHARE_EPOCH || conflict.GetShareEpoch() != 1 {
		t.Errorf("conflict = %v, want SHARE_EPOCH with epoch 1", conflict)
	}
	if got := len(s.owner.readAllChanges(s.vault)); got != 0 {
		t.Errorf("%d revisions stored, want none", got)
	}
	if _, accepted := s.commitShared("Food/milk.md", 1, []byte("milk")); !accepted.GetCommitted() {
		t.Error("a revision at the current epoch was not committed")
	}
}

func TestCommitRevisions_KeepsTheBlobOfTheParentWithoutAReservation(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	s.create()
	file := newID(16)
	first := s.owner.newFile(s.vault, file, nil, "Food/milk.md", []byte("milk"))
	s.owner.commit(s.vault, first)
	parent := syncprotocol.GetRevisionID(first)
	rewrap := &syncv1.Revision{
		VaultId: s.vault, FileId: file, ParentRev: parent[:], BlobId: first.GetBlobId(), BlobHash: first.GetBlobHash(), BlobSize: first.GetBlobSize(),
		EncMeta: []byte("share meta"), PathHash: first.GetPathHash(), ShareRootId: s.share, Epoch: 0, AuthorDeviceSignKey: s.owner.device.Public().Sign,
	}
	s.owner.signRevision(rewrap)

	response := s.owner.commit(s.vault, rewrap)

	if !response.GetCommitted() {
		t.Fatalf("response = %v, want a commit", response)
	}
	downloads, err := s.owner.blob.GetBlobUrls(t.Context(), s.vault, [][]byte{first.GetBlobId()})
	if err != nil || len(downloads) != 1 || !downloads[0].GetFound() {
		t.Errorf("downloads = %v, %v, want the blob still served for the new head", downloads, err)
	}
}

func TestCommitRevisions_RejectsRetainedBlobWithStaleParent(t *testing.T) {
	t.Parallel()
	s := newSharingHarness(t)
	file := newID(16)
	first := s.owner.newFile(s.vault, file, nil, "Food/milk.md", []byte("milk"))
	s.owner.commit(s.vault, first)
	stale := &syncv1.Revision{
		VaultId: s.vault, FileId: file, ParentRev: bytes.Repeat([]byte{4}, 32), BlobId: first.GetBlobId(), BlobHash: first.GetBlobHash(), BlobSize: first.GetBlobSize(),
		EncMeta: []byte("meta"), PathHash: first.GetPathHash(), AuthorDeviceSignKey: s.owner.device.Public().Sign,
	}
	s.owner.signRevision(stale)

	response := s.owner.commit(s.vault, stale)

	if response.GetCommitted() || len(response.GetConflicts()) != 1 || response.GetConflicts()[0].GetKind() != syncv1.ConflictKind_CONFLICT_KIND_PARENT_MISMATCH {
		t.Errorf("response = %v, want a PARENT_MISMATCH", response)
	}
}
