package plugin

import (
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/pluginset"
	"testing"
)

var errDiskFull = errors.New("disk full")

func TestFailedPersistRemovesTheTokenAndReportsTheError(t *testing.T) {
	host := newTokenHost(t)
	var reported []error
	host.Failures.SetErrorHandler(func(err error) { reported = append(reported, err) })
	host.Settings.SetSettings(map[string]map[string]any{})
	host.Settings.SetPersistHandler(func(string, string, any) error { return errDiskFull })

	if _, ok := host.Settings.GetPluginSettings(tokenPluginID)["token"]; ok {
		t.Errorf("a token that was not persisted is in force: %v", host.Settings.GetPluginSettings(tokenPluginID))
	}
	if len(reported) != 2 || !errors.Is(reported[0], errDiskFull) {
		t.Errorf("reported %v, want the failed write of token and secondToken", reported)
	}
}

func TestEnablingAPluginGeneratesItsToken(t *testing.T) {
	host := newTokenHost(t)
	writes := &tokenWrites{}
	loadTokenHost(host, map[string]map[string]any{tokenPluginID: {"enabled": false}}, writes)

	if err := updateSetting(t.Context(), host, tokenPluginID, pluginset.SettingEnabled, true); err != nil {
		t.Fatal(err)
	}

	got := writes.get(tokenPluginID, "token")
	if len(got) != 1 {
		t.Fatalf("token was written %v after the plugin was enabled, want once", got)
	}
	if token, ok := got[0].(string); !ok || !tokenPattern.MatchString(token) {
		t.Fatalf("the written token is %#v, want 64 lowercase hex characters", got[0])
	}
	if inForce := host.Settings.GetPluginSettings(tokenPluginID)["token"]; inForce != got[0] {
		t.Errorf("the token in force is %#v, want %#v", inForce, got[0])
	}
}
