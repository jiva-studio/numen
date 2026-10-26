package vaultwatch_test

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwatch"
)

// stopFileWatch follows a file of the vault and lets it go again.
func stopFileWatch(t *testing.T, path string) {
	t.Helper()
	ctx, stop := context.WithCancel(t.Context())
	moved, err := vaultwatch.Watcher{}.File(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	stop()
	for range moved {
	}
}

// A file followed inside the vault and let go leaves the vault followed.
func TestAVaultIsFollowedAfterAWatchInsideItStops(t *testing.T) {
	root := vaultfixture.CreateVault(t, map[string]string{"Note.md": "# Note\n", "settings/numen.json": "{}"}, nil)
	next := watchVault(t, root)

	stopFileWatch(t, filepath.Join(root, "settings", "numen.json"))
	write(t, root, "Later.md", "# Later\n")

	if got := next(); !slices.Contains(got, "Later.md") {
		t.Errorf("reported %v", got)
	}
}

// A file followed inside the vault before the vault was, and let go after, leaves the vault followed.
func TestAVaultIsFollowedAfterAnEarlierWatchInsideItStops(t *testing.T) {
	root := vaultfixture.CreateVault(t, map[string]string{"Note.md": "# Note\n", "settings/numen.json": "{}"}, nil)
	ctx, stop := context.WithCancel(t.Context())
	moved, err := vaultwatch.Watcher{}.File(ctx, filepath.Join(root, "settings", "numen.json"))
	if err != nil {
		t.Fatal(err)
	}
	next := watchVault(t, root)

	stop()
	for range moved {
	}
	write(t, root, "Later.md", "# Later\n")

	if got := next(); !slices.Contains(got, "Later.md") {
		t.Errorf("reported %v", got)
	}
}
