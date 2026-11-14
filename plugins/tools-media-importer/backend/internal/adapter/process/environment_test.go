//go:build unix

package process

import (
	"os"
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

	customOut, _, err := (Runner{}).Run(t.Context(), []string{"sh", "-c", "env"}, []string{"CUSTOM_VAR=pear", "PATH=" + os.Getenv("PATH")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(customOut), "CUSTOM_VAR=pear") || strings.Contains(string(customOut), "ORCHARD_TREE=") {
		t.Errorf("the program did not use the explicit environment:\n%s", customOut)
	}
}
