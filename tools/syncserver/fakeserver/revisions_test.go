package fakeserver_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func TestCommitRevisions_StoresAFileAndGetChangesReturnsItWithItsIDAndSeq(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	file := newID(16)
	revision := e.newFile(vault, file, nil, "Shopping list.md", []byte("body of the list"))

	response := e.commit(vault, revision)

	if !response.GetCommitted() || response.GetSeq() != 1 || len(response.GetRevisions()) != 1 {
		t.Fatalf("response = %v, want a commit at seq 1 with one revision", response)
	}
	id := syncprotocol.GetRevisionID(revision)
	if stored := response.GetRevisions()[0]; !bytes.Equal(stored.GetRevId(), id[:]) || stored.GetSeq() != 1 {
		t.Errorf("stored = %v, want the id the protocol computes and seq 1", stored)
	}
	changes, err := e.revision.GetChanges(t.Context(), vault, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes.GetChanges()) != 1 || !bytes.Equal(changes.GetChanges()[0].GetRevision().GetFileId(), file) ||
		changes.GetNextSeq() != 1 || changes.GetHeadSeq() != 1 || changes.GetHasMore() {
		t.Errorf("changes = %v, want the file with next 1, head 1 and no more", changes)
	}
}

func TestCommitRevisions_UsesOneSeqForABatchAndTheNextSeqForTheNextBatch(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	batch := e.commit(vault,
		e.newFile(vault, newID(16), nil, "milk.md", []byte("milk")),
		e.newFile(vault, newID(16), nil, "bread.md", []byte("bread")),
		e.newFile(vault, newID(16), nil, "eggs.md", []byte("eggs")),
	)
	next := e.commit(vault, e.newFile(vault, newID(16), nil, "tea.md", []byte("tea")))

	if batch.GetSeq() != 1 || next.GetSeq() != 2 {
		t.Errorf("seq of the batch = %d and of the next = %d, want 1 and 2", batch.GetSeq(), next.GetSeq())
	}
	for _, stored := range batch.GetRevisions() {
		if stored.GetSeq() != 1 {
			t.Errorf("a revision of the batch has seq %d, want 1", stored.GetSeq())
		}
	}
}

func TestCommitRevisions_AnswersAParentMismatchWithTheHeadAndUsesNoSeq(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	file := newID(16)
	first := e.newFile(vault, file, nil, "milk.md", []byte("milk"))
	e.commit(vault, first)
	stale := e.newFile(vault, file, nil, "milk.md", []byte("milk and bread"))

	response := e.commit(vault, stale)

	if response.GetCommitted() || len(response.GetConflicts()) != 1 {
		t.Fatalf("response = %v, want one conflict and no commit", response)
	}
	conflict := response.GetConflicts()[0]
	firstID := syncprotocol.GetRevisionID(first)
	if conflict.GetKind() != syncv1.ConflictKind_CONFLICT_KIND_PARENT_MISMATCH || !bytes.Equal(conflict.GetFileId(), file) ||
		!bytes.Equal(conflict.GetHead().GetRevId(), firstID[:]) {
		t.Errorf("conflict = %v, want a parent mismatch of the file that names its head", conflict)
	}
	next := e.commit(vault, e.newFile(vault, newID(16), nil, "bread.md", []byte("bread")))
	if next.GetSeq() != 2 {
		t.Errorf("seq after the conflict = %d, want 2: a conflict uses no seq", next.GetSeq())
	}
}

func TestCommitRevisions_AnswersAParentMismatchForAnUnknownFileWithNoHead(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	unknownParent := bytes.Repeat([]byte{0x33}, 32)

	response := e.commit(vault, e.newFile(vault, newID(16), unknownParent, "milk.md", []byte("milk")))

	if response.GetCommitted() || len(response.GetConflicts()) != 1 ||
		response.GetConflicts()[0].GetKind() != syncv1.ConflictKind_CONFLICT_KIND_PARENT_MISMATCH || response.GetConflicts()[0].GetHead() != nil {
		t.Errorf("response = %v, want a parent mismatch with no head", response)
	}
}

func TestCommitRevisions_AcceptsTheNextRevisionOfAFileAndFreesTheOldBlob(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	file := newID(16)
	first := e.newFile(vault, file, nil, "milk.md", []byte("milk"))
	e.commit(vault, first)
	firstID := syncprotocol.GetRevisionID(first)
	second := e.newFile(vault, file, firstID[:], "milk.md", []byte("milk and bread"))

	response := e.commit(vault, second)

	if !response.GetCommitted() || response.GetSeq() != 2 {
		t.Fatalf("response = %v, want a commit at seq 2", response)
	}
	downloads, err := e.blob.GetBlobUrls(t.Context(), vault, [][]byte{first.GetBlobId(), second.GetBlobId()})
	if err != nil {
		t.Fatal(err)
	}
	if len(downloads) != 2 || downloads[0].GetFound() || !downloads[1].GetFound() {
		t.Errorf("downloads = %v, want the old blob not found and the new one found", downloads)
	}
	changes := e.readAllChanges(vault)
	if len(changes) != 1 || changes[0].GetSeq() != 2 {
		t.Errorf("changes = %v, want only the head of the file at seq 2", changes)
	}
}

func TestCommitRevisions_AnswersPathTakenForAnotherFileOnTheSamePath(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	holder := newID(16)
	e.commit(vault, e.newFile(vault, holder, nil, "milk.md", []byte("milk")))

	response := e.commit(vault, e.newFile(vault, newID(16), nil, "milk.md", []byte("other milk")))

	if response.GetCommitted() || len(response.GetConflicts()) != 1 {
		t.Fatalf("response = %v, want one conflict and no commit", response)
	}
	conflict := response.GetConflicts()[0]
	if conflict.GetKind() != syncv1.ConflictKind_CONFLICT_KIND_PATH_TAKEN || !bytes.Equal(conflict.GetExistingFileId(), holder) ||
		!bytes.Equal(conflict.GetHead().GetRevision().GetFileId(), holder) {
		t.Errorf("conflict = %v, want path taken by the first file with its head", conflict)
	}
}

func TestCommitRevisions_StoresNothingOfABatchWithOneConflict(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	e.commit(vault, e.newFile(vault, newID(16), nil, "milk.md", []byte("milk")))
	good := e.newFile(vault, newID(16), nil, "bread.md", []byte("bread"))
	clash := e.newFile(vault, newID(16), nil, "milk.md", []byte("other milk"))

	response := e.commit(vault, good, clash)

	if response.GetCommitted() {
		t.Fatal("a batch with a conflict was committed")
	}
	changes := e.readAllChanges(vault)
	if len(changes) != 1 {
		t.Errorf("changes = %d, want only the first file: a batch is all or none", len(changes))
	}
	retry := e.commit(vault, good)
	if !retry.GetCommitted() || retry.GetSeq() != 2 {
		t.Errorf("the good revision alone: committed = %v at seq %d, want a commit at seq 2 with its blob still reserved", retry.GetCommitted(), retry.GetSeq())
	}
}

func TestCommitRevisions_TreatsTheSameBatchSentAgainAsAlreadyApplied(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	revision := e.newFile(vault, newID(16), nil, "milk.md", []byte("milk"))
	first := e.commit(vault, revision)

	again := e.commit(vault, revision)

	if !again.GetCommitted() || again.GetSeq() != first.GetSeq() || len(again.GetRevisions()) != 1 {
		t.Errorf("the retry = %v, want a commit at the seq of the first with the stored revision", again)
	}
	changes, err := e.revision.GetChanges(t.Context(), vault, 0, 0)
	if err != nil || changes.GetHeadSeq() != 1 || len(changes.GetChanges()) != 1 {
		t.Errorf("changes = %v, error = %v, want head seq 1 and one change: a retry takes no seq", changes, err)
	}
}

func TestCommitRevisions_RefusesARevisionThatIsNotSignedByADeviceOfTheAccount(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	stranger, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("milk")
	revision := e.newFile(vault, newID(16), nil, "milk.md", body)
	revision.AuthorDeviceSignKey = stranger.Public().Sign
	signature := stranger.SignForPlugin(syncprotocol.SignInPluginID, syncprotocol.GetRevisionMessage(revision))
	revision.Signature = signature[:]

	_, got := e.revision.CommitRevisions(t.Context(), vault, []*syncv1.Revision{revision})

	if connect.CodeOf(got) != connect.CodeInvalidArgument {
		t.Errorf("error = %v, want invalid argument", got)
	}
}

func TestCommitRevisions_RefusesARevisionWhoseBytesChangedAfterSigning(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	revision := e.newFile(vault, newID(16), nil, "milk.md", []byte("milk"))
	revision.EncMeta = []byte("changed meta")

	_, err := e.revision.CommitRevisions(t.Context(), vault, []*syncv1.Revision{revision})

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("error = %v, want invalid argument", err)
	}
}

func TestCommitRevisions_RefusesARevisionOfAnotherVaultThanTheRequestNames(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault, other := e.newVault(), e.newVault()
	revision := e.newFile(other, newID(16), nil, "milk.md", []byte("milk"))

	_, err := e.revision.CommitRevisions(t.Context(), vault, []*syncv1.Revision{revision})

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("error = %v, want invalid argument", err)
	}
}

func TestCommitRevisions_RefusesTwoRevisionsOfOneFileInOneBatch(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	file := newID(16)

	_, err := e.revision.CommitRevisions(t.Context(), vault, []*syncv1.Revision{
		e.newFile(vault, file, nil, "milk.md", []byte("milk")),
		e.newFile(vault, file, nil, "bread.md", []byte("bread")),
	})

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("error = %v, want invalid argument", err)
	}
}

func TestCommitRevisions_ReportsABlobThatWasNeverUploadedWithItsPositionInTheBatch(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	body := []byte("milk")
	uploaded := e.newFile(vault, newID(16), nil, "milk.md", body)
	missing := e.newRevisionOf(vault, newID(16), nil, "bread.md", e.reserveOnly(vault, 5), []byte("bread"))

	_, err := e.revision.CommitRevisions(t.Context(), vault, []*syncv1.Revision{uploaded, missing})

	var notReady syncprotocol.BlobNotReadyError
	if !errors.As(err, &notReady) || notReady.Index != 1 {
		t.Errorf("error = %v, want BlobNotReadyError at index 1", err)
	}
	if len(e.readAllChanges(vault)) != 0 {
		t.Error("a revision of the refused batch was stored")
	}
}

func TestCommitRevisions_RefusesABlobOfAnotherSizeThanTheOneReserved(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	body := []byte("milk")
	upload := e.reserveOnly(vault, 100)
	putErr := syncclient.PutBlob(t.Context(), http.DefaultClient, upload.GetUrl(), body)
	revision := e.newRevisionOf(vault, newID(16), nil, "milk.md", upload, body)

	_, err := e.revision.CommitRevisions(t.Context(), vault, []*syncv1.Revision{revision})

	var notReady syncprotocol.BlobNotReadyError
	if putErr == nil && !errors.As(err, &notReady) {
		t.Errorf("a body of 4 bytes for a reservation of 100: put error = %v, commit error = %v, want one of them to refuse it", putErr, err)
	}
}

func TestCommitRevisions_RefusesABlobThatAnotherRevisionAlreadyCommitted(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	body := []byte("milk")
	first := e.newFile(vault, newID(16), nil, "milk.md", body)
	e.commit(vault, first)
	reuse := e.newRevisionOf(vault, newID(16), nil, "bread.md", &syncv1.BlobUpload{BlobId: first.GetBlobId()}, body)

	_, err := e.revision.CommitRevisions(t.Context(), vault, []*syncv1.Revision{reuse})

	notReady, isNotReady := errors.AsType[syncprotocol.BlobNotReadyError](err)
	if !isNotReady || notReady.Index != 0 {
		t.Errorf("error = %v, want BlobNotReadyError of revision 0", err)
	}
}

func TestRevisionCallsOfAnotherAccountAreRefusedAsNoVault(t *testing.T) {
	t.Parallel()
	owner := newEnv(t)
	vault := owner.newVault()
	stranger := newSecondAccountOn(t, owner.address)
	revision := stranger.newFile(stranger.newVault(), newID(16), nil, "milk.md", []byte("milk"))
	revision.VaultId = vault
	stranger.signRevision(revision)

	_, commitErr := stranger.revision.CommitRevisions(t.Context(), vault, []*syncv1.Revision{revision})
	_, changesErr := stranger.revision.GetChanges(t.Context(), vault, 0, 0)
	_, reserveErr := stranger.blob.ReserveBlobs(t.Context(), vault, []uint64{10})
	_, urlsErr := stranger.blob.GetBlobUrls(t.Context(), vault, [][]byte{newID(16)})

	for name, err := range map[string]error{"commit": commitErr, "changes": changesErr, "reserve": reserveErr, "urls": urlsErr} {
		if !errors.Is(err, syncprotocol.ErrNoVault) {
			t.Errorf("%s: error = %v, want ErrNoVault", name, err)
		}
	}
}

func TestGetChanges_ReadsInPagesWithACursorAndNeverSplitsABatch(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	e.commit(vault, e.newFile(vault, newID(16), nil, "a.md", []byte("a")))
	e.commit(vault,
		e.newFile(vault, newID(16), nil, "b.md", []byte("b")),
		e.newFile(vault, newID(16), nil, "c.md", []byte("c")),
		e.newFile(vault, newID(16), nil, "d.md", []byte("d")),
	)
	e.commit(vault, e.newFile(vault, newID(16), nil, "e.md", []byte("e")))

	first, err := e.revision.GetChanges(t.Context(), vault, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	second, err := e.revision.GetChanges(t.Context(), vault, first.GetNextSeq(), 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(first.GetChanges()) != 4 || first.GetNextSeq() != 2 || !first.GetHasMore() || first.GetHeadSeq() != 3 {
		t.Errorf("first page = %d changes, next %d, more %v, head %d; want the first file and the whole batch of three, next 2, more, head 3",
			len(first.GetChanges()), first.GetNextSeq(), first.GetHasMore(), first.GetHeadSeq())
	}
	if len(second.GetChanges()) != 1 || second.GetNextSeq() != 3 || second.GetHasMore() {
		t.Errorf("second page = %d changes, next %d, more %v; want one change, next 3, no more", len(second.GetChanges()), second.GetNextSeq(), second.GetHasMore())
	}
}

func TestGetChanges_ReturnsAnEmptyPageWithTheCursorOfTheRequest(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	e.commit(vault, e.newFile(vault, newID(16), nil, "a.md", []byte("a")))

	page, err := e.revision.GetChanges(t.Context(), vault, 1, 0)

	if err != nil || len(page.GetChanges()) != 0 || page.GetNextSeq() != 1 || page.GetHasMore() || page.GetHeadSeq() != 1 {
		t.Errorf("page = %v, error = %v, want no changes, next 1, no more, head 1", page, err)
	}
}

func TestGetChanges_OrdersByTheSeqOfTheCommit(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	for _, name := range []string{"c.md", "a.md", "b.md"} {
		e.commit(vault, e.newFile(vault, newID(16), nil, name, []byte(name)))
	}

	changes := e.readAllChanges(vault)

	if len(changes) != 3 || changes[0].GetSeq() != 1 || changes[1].GetSeq() != 2 || changes[2].GetSeq() != 3 {
		t.Errorf("changes = %v, want seq 1, 2, 3 in order", changes)
	}
}

func TestReserveBlobs_ReturnsOneUploadForEachSizeInOrder(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()

	uploads, err := e.blob.ReserveBlobs(t.Context(), vault, []uint64{4096, 8192, 5})
	if err != nil {
		t.Fatal(err)
	}

	if len(uploads) != 3 || uploads[0].GetSize() != 4096 || uploads[1].GetSize() != 8192 || uploads[2].GetSize() != 5 {
		t.Fatalf("uploads = %v, want three in the order of the sizes", uploads)
	}
	seen := map[string]bool{}
	for _, upload := range uploads {
		if len(upload.GetBlobId()) != 16 || upload.GetUrl() == "" || seen[string(upload.GetBlobId())] {
			t.Errorf("upload = %v, want a new 16-byte id and an address", upload)
		}
		seen[string(upload.GetBlobId())] = true
	}
}

func TestReserveBlobs_RefusesASizeOrACountOutsideTheLimits(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	tooMany := make([]uint64, syncprotocol.MaxReserveBlobs+1)
	for i := range tooMany {
		tooMany[i] = 10
	}

	for name, sizes := range map[string][]uint64{
		"no size":        nil,
		"size zero":      {0},
		"over the limit": {syncprotocol.MaxBlobSize + 1},
		"too many":       tooMany,
	} {
		if _, err := e.blob.ReserveBlobs(t.Context(), vault, sizes); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("%s: error = %v, want invalid argument", name, err)
		}
	}
	if _, err := e.blob.ReserveBlobs(t.Context(), vault, []uint64{syncprotocol.MaxBlobSize}); err != nil {
		t.Errorf("a size at the limit: %v", err)
	}
}

func TestGetBlobUrls_ReturnsALinkOnlyForTheBlobOfACurrentHeadAndTheLinkReadsTheBody(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	body := []byte("stored bytes")
	revision := e.newFile(vault, newID(16), nil, "milk.md", body)
	reserved := e.reserveOnly(vault, 10)
	e.commit(vault, revision)

	downloads, err := e.blob.GetBlobUrls(t.Context(), vault, [][]byte{revision.GetBlobId(), reserved.GetBlobId(), newID(16)})
	if err != nil {
		t.Fatal(err)
	}

	if len(downloads) != 3 || !downloads[0].GetFound() || downloads[1].GetFound() || downloads[2].GetFound() {
		t.Fatalf("downloads = %v, want only the first found", downloads)
	}
	if downloads[1].GetUrl() != "" || downloads[2].GetUrl() != "" {
		t.Error("a blob that is not found has an address")
	}
	got, err := syncclient.GetBlob(t.Context(), http.DefaultClient, downloads[0].GetUrl())
	if err != nil || !bytes.Equal(got, body) {
		t.Errorf("body = %q, error = %v, want the uploaded body", got, err)
	}
}

func TestGetBlobUrls_RefusesMoreIDsThanTheLimit(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	ids := make([][]byte, syncprotocol.MaxBlobURLs+1)
	for i := range ids {
		ids[i] = newID(16)
	}

	_, err := e.blob.GetBlobUrls(t.Context(), vault, ids)

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("error = %v, want invalid argument", err)
	}
}

func TestTheAdminBlobsPathReturnsEveryStoredBodyAsRawBytes(t *testing.T) {
	t.Parallel()
	_, address := newServer(t)
	account := newOwnedAccount(t)
	device := account.addDevice(t, "First")
	e := newEnvOn(t, address, &account, device)
	vault := e.newVault()
	first, second := []byte("ciphertext of the first body"), []byte("ciphertext of the second body")
	e.upload(vault, first)
	e.upload(vault, second)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address+"/admin/blobs", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	})
	all, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusOK || !bytes.Contains(all, first) || !bytes.Contains(all, second) {
		t.Errorf("status = %d, %d bytes; want 200 with both stored bodies in it", response.StatusCode, len(all))
	}
}

type shiftedClock struct{ offset atomic.Int64 }

func (c *shiftedClock) Now() time.Time { return time.Now().Add(time.Duration(c.offset.Load())) }

func TestGetBlobUrls_ALinkStopsReadingTheBodyWhenItExpiresAndANewLinkReadsItAgain(t *testing.T) {
	t.Parallel()
	clock := &shiftedClock{}
	served := fakeserver.New(fakeserver.Config{Now: clock.Now})
	httpServer := httptest.NewServer(served)
	t.Cleanup(httpServer.Close)
	served.SetAudience(httpServer.URL)
	account := newOwnedAccount(t)
	e := newEnvOn(t, httpServer.URL, &account, account.addDevice(t, "First"))
	vault := e.newVault()
	body := []byte("stored bytes")
	revision := e.newFile(vault, newID(16), nil, "milk.md", body)
	e.commit(vault, revision)
	first, err := e.blob.GetBlobUrls(t.Context(), vault, [][]byte{revision.GetBlobId()})
	if err != nil {
		t.Fatal(err)
	}

	clock.offset.Store(int64(20 * time.Minute))
	_, expiredErr := syncclient.GetBlob(t.Context(), http.DefaultClient, first[0].GetUrl())
	second, err := e.blob.GetBlobUrls(t.Context(), vault, [][]byte{revision.GetBlobId()})
	if err != nil {
		t.Fatal(err)
	}
	got, renewedErr := syncclient.GetBlob(t.Context(), http.DefaultClient, second[0].GetUrl())

	if expiredErr == nil {
		t.Error("the link read the body after its expiry")
	}
	if renewedErr != nil || !bytes.Equal(got, body) {
		t.Errorf("body = %q, error = %v, want the stored body from the new link", got, renewedErr)
	}
}
