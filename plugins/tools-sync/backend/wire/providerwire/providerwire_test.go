package providerwire

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/vaultprovider"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
)

func TestJoinRemoteVaults_NamesTheLocalVaultOfAConnectedRemoteVault(t *testing.T) {
	t.Parallel()
	remote := []api.RemoteVault{
		{RemoteID: "aa", Name: "Work", IsConnected: true, SizeBytes: 1024},
		{RemoteID: "bb", Name: "Notes", IsConnected: false, SizeBytes: 7},
	}
	local := []api.Vault{
		{ID: "v-1", RemoteID: ""},
		{ID: "v-2", RemoteID: "aa"},
	}

	got := JoinRemoteVaults(remote, local)

	want := []vaultprovider.RemoteVault{
		{RemoteID: "aa", Name: "Work", SizeBytes: 1024, VaultID: "v-2"},
		{RemoteID: "bb", Name: "Notes", SizeBytes: 7, VaultID: ""},
	}
	if !slices.Equal(got, want) {
		t.Errorf("JoinRemoteVaults = %v, want %v", got, want)
	}
}
