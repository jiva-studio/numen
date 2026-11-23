package service_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/service"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/accountdata"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/session"
)

func newAccountBackend(t *testing.T) (*service.Backend, *world) {
	t.Helper()
	w := newWorld(t)
	keep := session.NewKeep(session.Deps{Identity: w.c.Account, Server: w.c.Sessions, Health: w.c.NewHealth(), Clock: w.c.Clock})
	deps := accountdata.Deps{Identity: w.c.Account, Challenges: w.c.Sessions, Account: w.c.Cloud, Deletion: keep}
	return service.NewBackend(w.c.SyncDeps()).WithAccountData(deps), w
}

func TestDeleteCloudData_StartsTheDeletionAndTheServerRefusesTheNextSignIn(t *testing.T) {
	backend, w := newAccountBackend(t)

	answer, err := backend.DeleteCloudData(t.Context(), api.DeleteCloudDataRequest{})

	if err != nil {
		t.Fatal(err)
	}
	if want := rfc3339(w.clock.Now().Add(fakeserver.DeletionGrace)); answer.DeleteAt != want {
		t.Errorf("deleteAt = %q, want %q: seven days from now", answer.DeleteAt, want)
	}
	if _, isDeleting := errors.AsType[sessiondomain.AccountDeletingError](w.c.SignIn(t.Context())); !isDeleting {
		t.Error("the sign-in of a device of a deleting account was not refused as deleting")
	}
}

func TestDeleteCloudData_LeavesTheNotesOfTheVaultsOnThisDevice(t *testing.T) {
	backend, w := newAccountBackend(t)
	w.c.Connect(t, "Test")
	w.c.WriteNote("Test", "a.md", "# Plan")

	if _, err := backend.DeleteCloudData(t.Context(), api.DeleteCloudDataRequest{}); err != nil {
		t.Fatal(err)
	}

	if text, isThere := w.c.ReadNote("Test", "a.md"); !isThere || text != "# Plan" {
		t.Errorf("note = %q, %v, want the text it had before the deletion", text, isThere)
	}
}

func TestDeleteCloudData_AsksNoServerWhenThisDeviceHasNoIdentity(t *testing.T) {
	backend, w := newAccountBackend(t)
	w.c.Fake.RemoveIdentity()

	_, err := backend.DeleteCloudData(t.Context(), api.DeleteCloudDataRequest{})

	if !errors.Is(err, sessiondomain.ErrNoIdentity) {
		t.Errorf("error = %v, want ErrNoIdentity", err)
	}
	if calls := w.server.Fake.CountCalls(syncv1connect.AccountServiceDeleteAccountProcedure); calls != 0 {
		t.Errorf("the server ran %d deletions, want none", calls)
	}
}

func TestGetAccountState_SaysDeletingWithTheDateOnceTheDeletionStarted(t *testing.T) {
	backend, _ := newAccountBackend(t)
	before, err := backend.GetAccountState(t.Context(), api.GetAccountStateRequest{})
	if err != nil || !before.HasIdentity || before.IsDeleting || before.Account == "" {
		t.Fatalf("state before = %+v, %v, want an identity that is not deleting", before, err)
	}
	deleted, err := backend.DeleteCloudData(t.Context(), api.DeleteCloudDataRequest{})
	if err != nil {
		t.Fatal(err)
	}

	after, err := backend.GetAccountState(t.Context(), api.GetAccountStateRequest{})

	if err != nil || !after.IsDeleting || after.DeleteAt != deleted.DeleteAt {
		t.Errorf("state after = %+v, %v, want deleting at %s", after, err, deleted.DeleteAt)
	}
}

func TestGetAccountState_ReportsNoIdentityWithoutAnError(t *testing.T) {
	backend, w := newAccountBackend(t)
	w.c.Fake.RemoveIdentity()

	got, err := backend.GetAccountState(t.Context(), api.GetAccountStateRequest{})

	if err != nil || got != (api.GetAccountStateResponse{}) {
		t.Errorf("state = %+v, %v, want an empty answer", got, err)
	}
}

func TestRestoreAccount_CancelsTheDeletionSoSignInWorksAgain(t *testing.T) {
	backend, w := newAccountBackend(t)
	if _, err := backend.DeleteCloudData(t.Context(), api.DeleteCloudDataRequest{}); err != nil {
		t.Fatal(err)
	}

	if _, err := backend.RestoreAccount(t.Context(), api.RestoreAccountRequest{}); err != nil {
		t.Fatalf("restore: %v", err)
	}

	if err := w.c.SignIn(t.Context()); err != nil {
		t.Errorf("sign-in after the restore: %v", err)
	}
	state, err := backend.GetAccountState(t.Context(), api.GetAccountStateRequest{})
	if err != nil || state.IsDeleting {
		t.Errorf("state = %+v, %v, want not deleting", state, err)
	}
}

func TestRestoreAccount_AfterTheGraceRanOutStartsAnEmptyAccount(t *testing.T) {
	backend, w := newAccountBackend(t)
	if _, err := backend.DeleteCloudData(t.Context(), api.DeleteCloudDataRequest{}); err != nil {
		t.Fatal(err)
	}
	w.clock.Advance(fakeserver.DeletionGrace + time.Hour)

	if err := w.c.SignIn(t.Context()); err != nil {
		t.Fatalf("sign-in after the grace: %v", err)
	}

	if _, err := w.c.Cloud.GetCatalog(t.Context(), 0); err != nil {
		t.Errorf("read the catalog of the new account: %v", err)
	}
}

func TestExportAccountData_ReturnsTheDocumentOfTheServer(t *testing.T) {
	backend, w := newAccountBackend(t)
	w.c.Connect(t, "Test")

	answer, err := backend.ExportAccountData(t.Context(), api.ExportAccountDataRequest{})

	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Account string `json:"account"`
		Vaults  []struct {
			ID string `json:"id"`
		} `json:"vaults"`
	}
	if err := json.Unmarshal([]byte(answer.Document), &document); err != nil {
		t.Fatalf("the document is not JSON: %v", err)
	}
	if document.Account == "" || len(document.Vaults) != 1 {
		t.Errorf("document = %+v, want the account and its one vault", document)
	}
}

func TestExportAccountData_AsksNoServerWhenThisDeviceHasNoIdentity(t *testing.T) {
	backend, w := newAccountBackend(t)
	w.c.Fake.RemoveIdentity()

	_, err := backend.ExportAccountData(t.Context(), api.ExportAccountDataRequest{})

	if !errors.Is(err, sessiondomain.ErrNoIdentity) {
		t.Errorf("error = %v, want ErrNoIdentity", err)
	}
	if calls := w.server.Fake.CountCalls(syncv1connect.AccountServiceExportAccountProcedure); calls != 0 {
		t.Errorf("the server ran %d exports, want none", calls)
	}
}

func TestGetSubscription_FailsAsDeletingForADeletingAccount(t *testing.T) {
	backend, _ := newAccountBackend(t)
	if _, err := backend.DeleteCloudData(t.Context(), api.DeleteCloudDataRequest{}); err != nil {
		t.Fatal(err)
	}

	_, err := backend.GetSubscription(t.Context(), api.GetSubscriptionRequest{})

	if _, isDeleting := errors.AsType[sessiondomain.AccountDeletingError](err); !isDeleting && !errors.Is(err, syncprotocol.ErrUnauthenticated) {
		t.Errorf("error = %v, want the refusal of a deleting account", err)
	}
}
