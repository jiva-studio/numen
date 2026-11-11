package app

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func windowOn(t *testing.T) (*editor.Installation, container.Config) {
	t.Helper()

	cfg := container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(t.TempDir(), "index.db"), RegistryPath: filepath.Join(t.TempDir(), "vaults.json")},
	})
	root := t.TempDir()
	if _, err := cfg.AddVault(root, "one"); err != nil {
		t.Fatal(err)
	}

	opened, err := container.OpenWindow(cfg, t.Context(), "one", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	})
	return opened, cfg
}

// TestTheWindowIsNamedAfterTheFileInFrontOfThePerson checks that the page
// reports which tab it is showing, and the window is named after the file of
// that tab.
func TestTheWindowIsNamedAfterTheFileInFrontOfThePerson(t *testing.T) {
	vault := domain.Vault{ID: "one", Name: "Notes"}
	for name, c := range map[string]struct {
		vault domain.Vault
		open  windowdomain.OpenTabs
		want  string
	}{
		"a note in front": {
			vault: vault,
			open: windowdomain.OpenTabs{
				ActiveID: "two",
				Tabs: []windowdomain.Tab{
					{ID: "one", Kind: "plex", Path: "Entropy.md"},
					{ID: "two", Kind: "note", Path: "Reading/Order.md"},
				},
			},
			want: "numen — Order.md",
		},
		"a tab holding no file": {
			vault: vault,
			open: windowdomain.OpenTabs{
				ActiveID: "one",
				Tabs:     []windowdomain.Tab{{ID: "one", Kind: "settings"}},
			},
			want: "numen — Notes",
		},
		"a window with nothing open": {vault: vault, want: "numen — Notes"},
		"a window standing on no vault": {
			open: windowdomain.OpenTabs{ActiveID: "one", Tabs: []windowdomain.Tab{{ID: "one", Kind: "files"}}},
			want: "numen",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := getWindowTitle(c.vault, c.open); got != c.want {
				t.Errorf("the window is called %q", got)
			}
		})
	}
}

func TestGetSecondaryWindowURL(t *testing.T) {
	cases := []struct {
		tab  windowdomain.DetachedTab
		want string
	}{
		{windowdomain.DetachedTab{ID: "plex:4f1c", X: 40}, "/?id=plex%3A4f1c&window=secondary"},
		{windowdomain.DetachedTab{}, "/?window=secondary"},
	}
	for _, c := range cases {
		if got := getSecondaryWindowURL(c.tab); got != c.want {
			t.Errorf("getSecondaryWindowURL(%+v) = %q, want %q", c.tab, got, c.want)
		}
	}
}
