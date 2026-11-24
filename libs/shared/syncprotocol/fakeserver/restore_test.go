package fakeserver_test

import (
	"bytes"
	"errors"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func newRestoreEnv(tb testing.TB) (*fakeserver.Server, *env, *syncclient.RestoreClient) {
	tb.Helper()
	server, address := newServer(tb)
	account := newOwnedAccount(tb)
	device := account.addDevice(tb, "First")
	e := newEnvOn(tb, address, &account, device)
	restore := syncclient.NewRestoreClient(address, http.DefaultClient, func() string { return e.token }, fakeserver.GetTestRoots(), time.Now)
	return server, e, restore
}

func readChangesResponse(tb testing.TB, e *env, vault []byte) *syncv1.GetChangesResponse {
	tb.Helper()
	response, err := e.revision.GetChanges(tb.Context(), vault, 0, 100)
	if err != nil {
		tb.Fatal(err)
	}
	return response
}

func TestGetRestoreEpoch_ReturnsNoEpochAndTheKeyBeforeTheFirstRestore(t *testing.T) {
	t.Parallel()
	_, e, restore := newRestoreEnv(t)
	vault := e.newVault()

	epoch, key, err := restore.GetRestoreEpoch(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if epoch != nil {
		t.Errorf("epoch = %v, want none before the first restore", epoch)
	}
	if len(key) != 32 {
		t.Errorf("epoch key has %d bytes, want 32", len(key))
	}
	if id := readChangesResponse(t, e, vault).GetRestoreEpochId(); len(id) != 0 {
		t.Errorf("restore_epoch_id = %x, want empty", id)
	}
}

func TestAnnounceRestoreEpoch_SignsAnEpochThatTheClientChecksAndGetChangesNamesIt(t *testing.T) {
	t.Parallel()
	server, e, restore := newRestoreEnv(t)
	vault := e.newVault()

	id, err := server.AnnounceRestoreEpoch()
	if err != nil {
		t.Fatal(err)
	}

	epoch, key, err := restore.GetRestoreEpoch(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if epoch == nil || !bytes.Equal(epoch.GetId(), id) {
		t.Fatalf("epoch = %v, want id %x", epoch, id)
	}
	if err := syncprotocol.CheckRestoreEpoch(key, epoch); err != nil {
		t.Errorf("CheckRestoreEpoch of the issued epoch = %v, want nil", err)
	}
	if got := readChangesResponse(t, e, vault).GetRestoreEpochId(); !bytes.Equal(got, id) {
		t.Errorf("restore_epoch_id = %x, want %x", got, id)
	}
}

func TestAnnounceRestoreEpoch_IssuesAnotherIDEachTimeAndTheLastOneIsCurrent(t *testing.T) {
	t.Parallel()
	server, e, _ := newRestoreEnv(t)
	vault := e.newVault()
	first, err := server.AnnounceRestoreEpoch()
	if err != nil {
		t.Fatal(err)
	}

	second, err := server.AnnounceRestoreEpoch()
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(first, second) {
		t.Error("two epochs have the same id")
	}
	if got := readChangesResponse(t, e, vault).GetRestoreEpochId(); !bytes.Equal(got, second) {
		t.Errorf("restore_epoch_id = %x, want the second %x", got, second)
	}
}

func TestRestore_GoesBackWithoutAnnouncingAnEpoch(t *testing.T) {
	t.Parallel()
	server, e, _ := newRestoreEnv(t)
	vault := e.newVault()
	file := newID(16)
	e.commit(vault, e.newFile(vault, file, nil, "a.md", []byte("one")))
	kept, err := server.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	e.commit(vault, e.newFile(vault, newID(16), nil, "b.md", []byte("two")))
	before := readChangesResponse(t, e, vault).GetHeadSeq()

	if err := server.Restore(kept); err != nil {
		t.Fatal(err)
	}

	after := readChangesResponse(t, e, vault)
	if after.GetHeadSeq() >= before {
		t.Errorf("head_seq after the restore = %d, want below %d", after.GetHeadSeq(), before)
	}
	if len(after.GetRestoreEpochId()) != 0 {
		t.Errorf("restore_epoch_id = %x, want empty: a restore announces nothing", after.GetRestoreEpochId())
	}
}

func TestDeleteAccountData_RemovesVaultsAndKeepsTheAccountSignIn(t *testing.T) {
	t.Parallel()
	server, e, _ := newRestoreEnv(t)
	vault := e.newVault()
	e.commit(vault, e.newFile(vault, newID(16), nil, "a.md", []byte("one")))

	if err := server.DeleteAccountData(); err != nil {
		t.Fatal(err)
	}

	if _, err := e.revision.GetChanges(t.Context(), vault, 0, 100); !isNoVault(err) {
		t.Errorf("GetChanges after the deletion = %v, want no such vault", err)
	}
	entries, err := e.catalog.GetCatalog(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("catalog holds %d entries after the deletion, want none", len(entries))
	}
	_ = signIn(t, e.address, e.device, e.account.chain)
}

func TestDeleteAccountData_AcceptsTheCatalogOfTheVaultSentAgain(t *testing.T) {
	t.Parallel()
	server, e, _ := newRestoreEnv(t)
	e.newVault()
	kept, err := e.catalog.GetCatalog(t.Context(), 0)
	if err != nil || len(kept) != 1 {
		t.Fatalf("catalog = %d entries, %v, want 1", len(kept), err)
	}
	if err := server.DeleteAccountData(); err != nil {
		t.Fatal(err)
	}

	response, err := e.catalog.AppendCatalogEntry(t.Context(), kept[0])

	if err != nil {
		t.Fatal(err)
	}
	if !response.GetCommitted() {
		t.Error("the entry that creates the vault again was not committed")
	}
}

func TestDropBlob_AnswersNotFoundWithTheMissingBlobOfAHead(t *testing.T) {
	t.Parallel()
	server, e, _ := newRestoreEnv(t)
	vault := e.newVault()
	revision := e.newFile(vault, newID(16), nil, "a.md", []byte("one"))
	e.commit(vault, revision)
	if err := server.DropBlob(revision.GetBlobId()); err != nil {
		t.Fatal(err)
	}

	_, err := e.blob.GetBlobUrls(t.Context(), vault, [][]byte{revision.GetBlobId()})

	missing, ok := asBlobMissing(err)
	if !ok || len(missing.IDs) != 1 || !bytes.Equal(missing.IDs[0], revision.GetBlobId()) {
		t.Fatalf("GetBlobUrls = %v, want BlobMissingError naming the blob", err)
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("code = %v, want not found", connect.CodeOf(err))
	}
}

func isNoVault(err error) bool {
	return errors.Is(err, syncprotocol.ErrNoVault)
}

func asBlobMissing(err error) (syncprotocol.BlobMissingError, bool) {
	return errors.AsType[syncprotocol.BlobMissingError](err)
}
