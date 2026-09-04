package main

import (
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/webui"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// windowOn is a window open on a vault of this test's own.
func windowOn(t *testing.T) (*webui.Installation, container.Config) {
	t.Helper()

	cfg := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
	}
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	add := vaults.Add{Identity: cfg.VaultIdentity(), Registry: registry, Now: time.Now}
	if _, err := add.Execute(root, "one"); err != nil {
		t.Fatal(err)
	}

	opened, err := webui.Open(t.Context(), cfg, "one", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { opened.Close() })
	return opened, cfg
}

// TestTheWindowIsNamedAfterTheFileInFrontOfThePerson. The page says which tab
// it is showing, and the window is named after the file that tab holds.
func TestTheWindowIsNamedAfterTheFileInFrontOfThePerson(t *testing.T) {
	vault := domain.Vault{ID: "one", Name: "Notes"}
	for name, c := range map[string]struct {
		vault domain.Vault
		open  domain.OpenTabs
		want  string
	}{
		"a note in front": {
			vault: vault,
			open: domain.OpenTabs{
				FrontID: "two",
				Tabs: []domain.Tab{
					{ID: "one", Kind: domain.TabPlex, Path: "Entropy.md"},
					{ID: "two", Kind: domain.TabNote, Path: "Reading/Order.md"},
				},
			},
			want: "numen — Order.md",
		},
		"a tab holding no file": {
			vault: vault,
			open: domain.OpenTabs{
				FrontID: "one",
				Tabs:    []domain.Tab{{ID: "one", Kind: "settings"}},
			},
			want: "numen — Notes",
		},
		"a window with nothing open": {vault: vault, want: "numen — Notes"},
		"a window standing on no vault": {
			open: domain.OpenTabs{FrontID: "one", Tabs: []domain.Tab{{ID: "one", Kind: "files"}}},
			want: "numen",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := titled(c.vault, c.open); got != c.want {
				t.Errorf("the window is called %q", got)
			}
		})
	}
}
