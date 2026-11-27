package service_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
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

func TestDeleteCloudData_DeletesTheAccountAndTheServerRefusesTheNextSignIn(t *testing.T) {
	backend, w := newAccountBackend(t)

	if _, err := backend.DeleteCloudData(t.Context(), api.DeleteCloudDataRequest{}); err != nil {
		t.Fatal(err)
	}

	if err := w.c.SignIn(t.Context()); !errors.Is(err, sessiondomain.ErrUnauthenticated) {
		t.Errorf("sign-in after the deletion = %v, want ErrUnauthenticated", err)
	}
}

func TestDeleteCloudData_KeepsTheNotesOfTheVaultsOnThisDevice(t *testing.T) {
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
	if calls := w.server.CountCalls(syncv1connect.AccountServiceDeleteAccountProcedure); calls != 0 {
		t.Errorf("the server ran %d deletions, want none", calls)
	}
}

func TestGetAccountState_ReturnsDeletedOnceThisDeviceDeletedTheCloudData(t *testing.T) {
	backend, _ := newAccountBackend(t)
	before, err := backend.GetAccountState(t.Context(), api.GetAccountStateRequest{})
	if err != nil || !before.HasIdentity || before.IsDeleted || before.Account == "" {
		t.Fatalf("state before = %+v, %v, want an identity whose cloud data is not deleted", before, err)
	}
	if _, err := backend.DeleteCloudData(t.Context(), api.DeleteCloudDataRequest{}); err != nil {
		t.Fatal(err)
	}

	after, err := backend.GetAccountState(t.Context(), api.GetAccountStateRequest{})

	if err != nil || !after.IsDeleted || after.Account != before.Account {
		t.Errorf("state after = %+v, %v, want the cloud data of %s deleted", after, err, before.Account)
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
	if calls := w.server.CountCalls(syncv1connect.AccountServiceExportAccountProcedure); calls != 0 {
		t.Errorf("the server ran %d exports, want none", calls)
	}
}

func TestGetSubscription_FailsForADeletedAccount(t *testing.T) {
	backend, _ := newAccountBackend(t)
	if _, err := backend.DeleteCloudData(t.Context(), api.DeleteCloudDataRequest{}); err != nil {
		t.Fatal(err)
	}

	_, err := backend.GetSubscription(t.Context(), api.GetSubscriptionRequest{})

	if !errors.Is(err, sessiondomain.ErrUnauthenticated) && !errors.Is(err, syncprotocol.ErrUnauthenticated) {
		t.Errorf("error = %v, want the refusal of a deleted account", err)
	}
}
