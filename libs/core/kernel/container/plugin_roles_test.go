package container

import "testing"

func openPluginRoles(tb testing.TB, target string, dirs ...string) *PluginRoles {
	tb.Helper()
	host, err := OpenPluginRoles(target, dirs...)
	if err != nil {
		tb.Logf("discover plugins: %v", err)
	}
	return host
}
