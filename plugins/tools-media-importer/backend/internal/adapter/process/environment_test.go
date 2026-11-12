//go:build unix

package process

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

func TestRunWithoutAnEnvironmentHidesTheHostVariablesFromTheProgram(t *testing.T) {
	t.Setenv(hostcall.PluginTokenEnv, "orchard-token")
	t.Setenv("ORCHARD_TREE", "apple")

	out, _, err := (Runner{}).Run(t.Context(), []string{"sh", "-c", "env"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(out), hostcall.PluginTokenEnv+"=") {
		t.Errorf("the program saw the plugin token:\n%s", out)
	}
	if !strings.Contains(string(out), "ORCHARD_TREE=apple") {
		t.Errorf("the program lost an ordinary variable:\n%s", out)
	}
}
