package container_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

func newExternalPluginHost(t *testing.T, target string, dirs ...string) *container.PluginHost {
	t.Helper()
	host, err := container.NewPluginHost(container.Config{}, func(error) {}, target, dirs...)
	if err != nil {
		t.Fatal(err)
	}
	return host
}
