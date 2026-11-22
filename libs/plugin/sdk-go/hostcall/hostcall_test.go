package hostcall_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

func TestTheHostIsWhereTheVariablesSayAndTheTokenIsTheirs(t *testing.T) {
	endpoint, err := hostcall.FromVars(map[string]string{hostcall.HostPortEnv: " 4321 ", hostcall.PluginTokenEnv: "tok"})

	if err != nil || endpoint.BaseURL != "http://127.0.0.1:4321" || endpoint.Token != "tok" {
		t.Errorf("got %+v (%v)", endpoint, err)
	}
}

func TestNoPortIsNoHost(t *testing.T) {
	if _, err := hostcall.FromVars(map[string]string{}); !errors.Is(err, hostcall.ErrNoHost) {
		t.Errorf("got %v", err)
	}
	t.Setenv(hostcall.HostPortEnv, "")
	if _, err := hostcall.FromEnv(); !errors.Is(err, hostcall.ErrNoHost) {
		t.Errorf("got %v", err)
	}
}

func TestTheEnvironmentNamesTheHost(t *testing.T) {
	t.Setenv(hostcall.HostPortEnv, "99")
	t.Setenv(hostcall.PluginTokenEnv, "t")

	endpoint, err := hostcall.FromEnv()

	if err != nil || endpoint.BaseURL != "http://127.0.0.1:99" || endpoint.Token != "t" {
		t.Errorf("got %+v (%v)", endpoint, err)
	}
}
