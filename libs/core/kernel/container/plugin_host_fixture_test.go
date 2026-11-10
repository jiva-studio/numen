package container

import "testing"

func newPluginHost(t *testing.T, target string, dirs ...string) *PluginHost {
	t.Helper()
	host, err := NewPluginHost(target, dirs...)
	if err != nil {
		t.Fatal(err)
	}
	return host
}
