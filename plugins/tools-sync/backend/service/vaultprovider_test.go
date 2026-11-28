package service

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/errorwire"
)

func TestVaultProvider_ListsNoVaultsWhileNoServerIsSet(t *testing.T) {
	t.Parallel()
	provider := newVaultProvider(newRuntime(parts{clock: clock.System{}}))

	listed, err := provider.ListRemoteVaults(t.Context())
	if err != nil {
		t.Fatalf("ListRemoteVaults failed: %v", err)
	}
	if listed == nil || len(listed) != 0 {
		t.Errorf("ListRemoteVaults = %v, want an empty list", listed)
	}
}

func TestVaultProvider_ConnectKeepsTheProductTextOfTheFailure(t *testing.T) {
	t.Parallel()
	provider := newVaultProvider(newRuntime(parts{clock: clock.System{}}))

	err := provider.ConnectRemoteVault(t.Context(), "ab12", domain.VaultID("v-1"))
	if err == nil {
		t.Fatal("ConnectRemoteVault succeeded without a server")
	}
	if want := errorwire.GetText(errNotConfigured); !strings.Contains(err.Error(), want) {
		t.Errorf("ConnectRemoteVault failed with %q, want it to contain %q", err.Error(), want)
	}
}
