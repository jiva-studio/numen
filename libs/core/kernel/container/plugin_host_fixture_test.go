package container

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/indexfolder"
)

func newPluginHost(t *testing.T, target string, dirs ...string) *TestPlugins {
	t.Helper()
	execution, preferences, endpoints, err := NewPluginHost("", NewResources(), func(error) {}, target, dirs...)
	if err != nil {
		t.Fatal(err)
	}
	return &TestPlugins{Execution: execution, Preferences: preferences, Endpoints: endpoints}
}

func prepareIndexPath(c vaultParts) (string, error) { return indexfolder.Prepare(c.IndexPath) }
