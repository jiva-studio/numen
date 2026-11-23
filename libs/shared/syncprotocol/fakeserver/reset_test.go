package fakeserver_test

import (
	"net/http"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func (e *env) newRemoval(vault, file, parent []byte, path string) *syncv1.Revision {
	return e.signRevision(&syncv1.Revision{
		VaultId: vault, FileId: file, ParentRev: parent, Deleted: true,
		EncMeta: []byte("meta of " + path), PathHash: hashOf([]byte(path)), AuthorDeviceSignKey: e.device.Public().Sign,
	})
}

func (e *env) removeNote(vault []byte, path string) {
	e.tb.Helper()
	file := newID(16)
	first := e.newFile(vault, file, nil, path, []byte("text of "+path))
	stored := e.commit(vault, first).GetRevisions()[0]
	e.commit(vault, e.newRemoval(vault, file, stored.GetRevId(), path))
}

func newEnvWithServer(tb testing.TB) (*env, *fakeserver.Server) {
	tb.Helper()
	server, address := newServer(tb)
	account := newOwnedAccount(tb)
	return newEnvOn(tb, address, &account, account.addDevice(tb, "First")), server
}

func TestGetChanges_AnswersWithAResetAndEveryLiveFileOnceTheServerForgotTheRemovals(t *testing.T) {
	t.Parallel()
	e, server := newEnvWithServer(t)
	vault := e.newVault()
	kept := e.commit(vault, e.newFile(vault, newID(16), nil, "Plan.md", []byte("tea"))).GetRevisions()[0]
	e.removeNote(vault, "Shopping list.md")
	server.ForgetRemovals()

	changes, err := e.revision.GetChanges(t.Context(), vault, kept.GetSeq(), 0)
	if err != nil {
		t.Fatal(err)
	}

	if !changes.GetIsReset() {
		t.Error("is_reset = false, want true for a position older than the forgotten removals")
	}
	if len(changes.GetChanges()) != 1 || string(changes.GetChanges()[0].GetRevId()) != string(kept.GetRevId()) || changes.GetChanges()[0].GetRevision().GetDeleted() {
		t.Errorf("changes = %v, want the one live file from the start and no removal", changes.GetChanges())
	}
}

func TestGetChanges_AnswersWithNoResetForAPositionAfterTheForgottenRemovals(t *testing.T) {
	t.Parallel()
	e, server := newEnvWithServer(t)
	vault := e.newVault()
	e.removeNote(vault, "Shopping list.md")
	server.ForgetRemovals()
	added := e.commit(vault, e.newFile(vault, newID(16), nil, "Plan.md", []byte("tea"))).GetRevisions()[0]

	changes, err := e.revision.GetChanges(t.Context(), vault, added.GetSeq(), 0)
	if err != nil {
		t.Fatal(err)
	}

	if changes.GetIsReset() || len(changes.GetChanges()) != 0 {
		t.Errorf("changes = %v, want no reset and no change", changes)
	}
}

func TestGetChanges_AnswersWithNoResetBeforeTheServerForgetsAnything(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	e.removeNote(vault, "Shopping list.md")

	changes, err := e.revision.GetChanges(t.Context(), vault, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if changes.GetIsReset() || len(changes.GetChanges()) != 1 || !changes.GetChanges()[0].GetRevision().GetDeleted() {
		t.Errorf("changes = %v, want no reset and the removal as a change", changes)
	}
}

func TestAdminForgetRemovals_MakesThePositionBeforeItAResetOverHTTP(t *testing.T) {
	t.Parallel()
	_, address := newServer(t)
	account := newOwnedAccount(t)
	e := newEnvOn(t, address, &account, account.addDevice(t, "First"))
	vault := e.newVault()
	kept := e.commit(vault, e.newFile(vault, newID(16), nil, "Plan.md", []byte("tea"))).GetRevisions()[0]
	e.removeNote(vault, "Shopping list.md")
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, address+"/admin/forget-removals", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}

	changes, err := e.revision.GetChanges(t.Context(), vault, kept.GetSeq(), 0)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusNoContent || !changes.GetIsReset() {
		t.Errorf("status = %d, is_reset = %v, want 204 and true", response.StatusCode, changes.GetIsReset())
	}
}
