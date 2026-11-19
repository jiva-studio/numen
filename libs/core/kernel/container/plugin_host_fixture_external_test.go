package container_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

func newExternalPluginHost(t *testing.T, target string, dirs ...string) *container.TestPlugins {
	t.Helper()
	execution, preferences, endpoints, err := container.NewPluginHost("", container.NewResources(), func(error) {}, target, dirs...)
	if err != nil {
		t.Fatal(err)
	}
	return &container.TestPlugins{Execution: execution, Preferences: preferences, Endpoints: endpoints}
}
