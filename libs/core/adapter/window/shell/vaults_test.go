package shell

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// newVaultsClient is the list of vaults the window answers with.
func newVaultsClient(t *testing.T, api *API) numenv1connect.VaultsServiceClient {
	t.Helper()
	server := httptest.NewServer(api.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	return numenv1connect.NewVaultsServiceClient(server.Client(), server.URL)
}

func TestTheWindowListsTheVaultsOfTheInstallation(t *testing.T) {
	api := newAPI(t)
	api.ListVaults = func() ([]domain.Vault, error) {
		return []domain.Vault{
			{ID: "v1", Name: "Studies", Path: "/vaults/Studies"},
			{ID: "v2", Name: "Words", Path: "/vaults/Words"},
		}, nil
	}

	response, err := newVaultsClient(t, api).ListVaults(t.Context(), connect.NewRequest(&v1.ListVaultsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	got := response.Msg.GetVaults()
	if len(got) != 2 || got[0].GetId() != "v1" || got[0].GetName() != "Studies" ||
		got[0].GetPath() != "/vaults/Studies" || got[1].GetId() != "v2" {
		t.Fatalf("the vaults are %v", got)
	}
}

func TestAListThatCannotBeReadIsAnError(t *testing.T) {
	api := newAPI(t)
	api.ListVaults = func() ([]domain.Vault, error) { return nil, errors.New("the list is locked") }

	_, err := newVaultsClient(t, api).ListVaults(t.Context(), connect.NewRequest(&v1.ListVaultsRequest{}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("the error is %v", err)
	}
}

func TestAWindowWithNoListServesNone(t *testing.T) {
	client := newVaultsClient(t, newAPI(t))

	_, err := client.ListVaults(t.Context(), connect.NewRequest(&v1.ListVaultsRequest{}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("the error is %v", err)
	}
	_, err = client.AddVault(t.Context(), connect.NewRequest(&v1.AddVaultRequest{Path: "/vaults/New"}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("adding a vault answered %v", err)
	}
}
