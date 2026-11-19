package container

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

func openPluginRoles(tb testing.TB, target string, dirs ...string) *testPluginRoles {
	tb.Helper()
	host := newTestPluginRoles(GetClock(NewResources()), target, dirs...)
	err := InitializePluginRoles("", NewResources(), host.Management, host.Supervision, func(error) {})
	if err != nil {
		tb.Logf("discover plugins: %v", err)
	}
	return host
}

type testPluginRoles struct {
	Management  *PluginManagement
	Supervision *PluginSupervision
}

func newTestPluginRoles(clock machineport.Clock, target string, dirs ...string) *testPluginRoles {
	management, supervision := NewPluginRoles(clock, target, dirs...)
	return &testPluginRoles{Management: management, Supervision: supervision}
}
