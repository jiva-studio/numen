package vaultaddress_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/vaultaddress"
)

func TestFindMatchesIdentifierThenNameThenFolder(t *testing.T) {
	entries := []host.VaultEntry{
		{Vault: domain.Vault{ID: "v-tide", Name: "garden", Path: "/vaults/tide"}, IsMissing: false},
		{Vault: domain.Vault{ID: "v-garden", Name: "tide", Path: "/vaults/garden"}, IsMissing: false},
	}
	for address, want := range map[string]domain.VaultID{
		"v-garden":        "v-garden",
		"garden":          "v-tide",
		"/vaults/garden/": "v-garden",
	} {
		got, ok := vaultaddress.Find(entries, address)
		if !ok || got.ID != want {
			t.Errorf("Find(%q) = %q, %v, want %q", address, got.ID, ok, want)
		}
	}
	if _, ok := vaultaddress.Find(entries, "meadow"); ok {
		t.Error("Find(meadow) matched a vault")
	}
}
