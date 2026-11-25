package vaultprovider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

type recorder struct {
	vaults    []RemoteVault
	connected []string
	err       error
}

func (r *recorder) ListRemoteVaults(context.Context) ([]RemoteVault, error) {
	return r.vaults, r.err
}

func (r *recorder) ConnectRemoteVault(_ context.Context, remoteID string, vaultID domain.VaultID) error {
	r.connected = append(r.connected, remoteID+" "+string(vaultID))
	return r.err
}

func serve(t *testing.T, h Handler) numenv1connect.VaultProviderServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(NewHandler(h))
	web := httptest.NewServer(mux)
	t.Cleanup(web.Close)
	return numenv1connect.NewVaultProviderServiceClient(web.Client(), web.URL)
}

func TestTheHandlerListsTheRemoteVaultsOfThePlugin(t *testing.T) {
	plugin := &recorder{vaults: []RemoteVault{
		{RemoteID: "r-1", Name: "Notes", SizeBytes: 2048, VaultID: "v-1"},
		{RemoteID: "r-2", Name: "Books"},
	}}
	client := serve(t, plugin)

	got, err := client.ListRemoteVaults(t.Context(), connect.NewRequest(&v1.ListRemoteVaultsRequest{}))

	if err != nil {
		t.Fatal(err)
	}
	vaults := got.Msg.GetVaults()
	if len(vaults) != 2 || vaults[0].GetRemoteId() != "r-1" || vaults[0].GetName() != "Notes" ||
		vaults[0].GetSizeBytes() != 2048 || vaults[0].GetVaultId() != "v-1" || vaults[1].GetVaultId() != "" {
		t.Errorf("the client got %v", vaults)
	}
}

func TestTheHandlerPassesTheRemoteVaultAndTheVaultToThePlugin(t *testing.T) {
	plugin := &recorder{}
	client := serve(t, plugin)

	_, err := client.ConnectRemoteVault(t.Context(), connect.NewRequest(&v1.ConnectRemoteVaultRequest{RemoteId: "r-1", VaultId: "v-9"}))

	if err != nil {
		t.Fatal(err)
	}
	if len(plugin.connected) != 1 || plugin.connected[0] != "r-1 v-9" {
		t.Errorf("the plugin was called with %q", plugin.connected)
	}
}

func TestAFailureOfThePluginReachesTheClientWithItsCodeAndMessage(t *testing.T) {
	plugin := &recorder{err: rpcerror.New(rpcerror.FailedPrecondition, errors.New("The vault is connected already"))}
	client := serve(t, plugin)

	_, err := client.ConnectRemoteVault(t.Context(), connect.NewRequest(&v1.ConnectRemoteVaultRequest{RemoteId: "r-1", VaultId: "v-1"}))

	var failure *connect.Error
	if !errors.As(err, &failure) || failure.Code() != connect.CodeFailedPrecondition || failure.Message() != "The vault is connected already" {
		t.Errorf("the call failed with %v", err)
	}
}
