package hostcall_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

func TestTheChildEnvironmentLacksTheHostVariables(t *testing.T) {
	for _, name := range []string{
		hostcall.HostPortEnv, hostcall.PluginTokenEnv, hostcall.PluginPortEnv, hostcall.PluginIDEnv, hostcall.VaultPathEnv,
	} {
		t.Setenv(name, "orchard-value")
	}
	t.Setenv("NUMEN_PLUGIN_TOKEN_SUFFIXED", "kept")
	t.Setenv("ORCHARD_TREE", "apple")

	env := hostcall.GetChildEnvironment()

	for _, entry := range env {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "NUMEN_") && name != "NUMEN_PLUGIN_TOKEN_SUFFIXED" {
			t.Errorf("the child environment holds %q", entry)
		}
	}
	for _, want := range []string{"ORCHARD_TREE=apple", "NUMEN_PLUGIN_TOKEN_SUFFIXED=kept"} {
		if !slices.Contains(env, want) {
			t.Errorf("the child environment lacks %q", want)
		}
	}
}
