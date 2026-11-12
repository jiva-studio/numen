package hostcall_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

func listHostEnvNames() []string {
	return []string{
		hostcall.HostPortEnv, hostcall.PluginTokenEnv, hostcall.PluginPortEnv, hostcall.PluginIDEnv, hostcall.VaultPathEnv,
	}
}

func TestTheChildEnvironmentLacksTheHostVariables(t *testing.T) {
	for _, name := range listHostEnvNames() {
		t.Setenv(name, "orchard-value")
	}
	t.Setenv("NUMEN_PLUGIN_TOKEN_SUFFIXED", "kept")
	t.Setenv("ORCHARD_TREE", "apple")

	env := hostcall.GetChildEnvironment()

	for _, entry := range env {
		name, _, _ := strings.Cut(entry, "=")
		if slices.Contains(listHostEnvNames(), name) {
			t.Errorf("the child environment holds %q", entry)
		}
	}
	for _, want := range []string{"ORCHARD_TREE=apple", "NUMEN_PLUGIN_TOKEN_SUFFIXED=kept"} {
		if !slices.Contains(env, want) {
			t.Errorf("the child environment lacks %q", want)
		}
	}
}
