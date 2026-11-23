package fakeserver_test

import (
	"net/http"
	"testing"
)

func postAdmin(tb testing.TB, address, path string) {
	tb.Helper()
	request, err := http.NewRequestWithContext(tb.Context(), http.MethodPost, address+path, nil)
	if err != nil {
		tb.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		tb.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			tb.Error(err)
		}
	}()
	if response.StatusCode >= 300 {
		tb.Fatalf("POST %s answered %d", path, response.StatusCode)
	}
}

func TestAdminBackupAndRestore_PutTheServerBackToTheBackupWithoutAnnouncingIt(t *testing.T) {
	t.Parallel()
	_, e, _ := newRestoreEnv(t)
	vault := e.newVault()
	postAdmin(t, e.address, "/admin/backup")
	e.commit(vault, e.newFile(vault, newID(16), nil, "a.md", []byte("one")))

	postAdmin(t, e.address, "/admin/restore")

	response := readChangesResponse(t, e, vault)
	if len(response.GetChanges()) != 0 || len(response.GetRestoreEpochId()) != 0 {
		t.Errorf("changes %d epoch %x, want the state of the backup and no epoch", len(response.GetChanges()), response.GetRestoreEpochId())
	}
}

func TestAdminAnnounce_IssuesAnEpochTheClientReads(t *testing.T) {
	t.Parallel()
	_, e, restore := newRestoreEnv(t)
	e.newVault()

	postAdmin(t, e.address, "/admin/announce")

	epoch, _, err := restore.GetRestoreEpoch(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if epoch == nil || len(epoch.GetId()) != 16 {
		t.Errorf("epoch = %v, want one of 16 bytes", epoch)
	}
}

func TestAdminRetention_DeletesTheVaultsAndKeepsTheAccount(t *testing.T) {
	t.Parallel()
	_, e, _ := newRestoreEnv(t)
	vault := e.newVault()

	postAdmin(t, e.address, "/admin/retention")

	if _, err := e.revision.GetChanges(t.Context(), vault, 0, 100); !isNoVault(err) {
		t.Errorf("GetChanges after the retention = %v, want no such vault", err)
	}
	_ = signIn(t, e.address, e.device, e.account.chain)
}
