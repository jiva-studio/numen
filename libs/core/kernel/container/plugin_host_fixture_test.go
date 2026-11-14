package container

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/indexfolder"
)

func newPluginHost(t *testing.T, target string, dirs ...string) *PluginHost {
	t.Helper()
	host, err := NewPluginHost(Config{}, func(error) {}, target, dirs...)
	if err != nil {
		t.Fatal(err)
	}
	return host
}

func prepareIndexPath(c vaultWiring) (string, error) { return indexfolder.Prepare(c.IndexPath) }
