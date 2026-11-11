package vaultwatch_test

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/watchhub"
)

func stopFileWatch(t *testing.T, hub *watchhub.Hub, path string) {
	t.Helper()
	ctx, stop := context.WithCancel(t.Context())
	moved, err := vaultwatch.Watcher{Hub: hub}.File(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	stop()
	for range moved {
	}
}

func getNestedFixture() map[string]string {
	return map[string]string{"Note.md": "# Note\n", "settings/numen.json": "{}"}
}

func nextAfterFixture(next func() []string) []string {
	for {
		got := next()
		if slices.ContainsFunc(got, func(path string) bool { _, isFixture := getNestedFixture()[path]; return !isFixture }) {
			return got
		}
	}
}

// A file followed inside the vault and released leaves the vault followed.
func TestAVaultIsFollowedAfterAWatchInsideItStops(t *testing.T) {
	root := vaultfixture.CreateVault(t, getNestedFixture(), nil)
	// The two watches go through one hub, as every watch of one container does.
	hub := watchhub.NewHub()
	next := watchVault(t, hub, root)

	stopFileWatch(t, hub, filepath.Join(root, "settings", "numen.json"))
	write(t, root, "Later.md", "# Later\n")

	if got := nextAfterFixture(next); !slices.Contains(got, "Later.md") {
		t.Errorf("reported %v", got)
	}
}

// A file followed inside the vault before the vault was, and released after, leaves the vault followed.
func TestAVaultIsFollowedAfterAnEarlierWatchInsideItStops(t *testing.T) {
	root := vaultfixture.CreateVault(t, getNestedFixture(), nil)
	// The two watches go through one hub, as every watch of one container does.
	hub := watchhub.NewHub()
	ctx, stop := context.WithCancel(t.Context())
	moved, err := vaultwatch.Watcher{Hub: hub}.File(ctx, filepath.Join(root, "settings", "numen.json"))
	if err != nil {
		t.Fatal(err)
	}
	next := watchVault(t, hub, root)

	stop()
	for range moved {
	}
	write(t, root, "Later.md", "# Later\n")

	if got := nextAfterFixture(next); !slices.Contains(got, "Later.md") {
		t.Errorf("reported %v", got)
	}
}
