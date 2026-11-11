package hostserver

import "testing"

func TestParseSettingsTrimsTheToken(t *testing.T) {
	if got := ParseSettings(nil); got.Token != "" {
		t.Errorf("ParseSettings without values = %+v, want an empty token", got)
	}
	if got := ParseSettings(map[string]string{"port": "9002", "token": " map-token "}); got.Token != "map-token" {
		t.Errorf("ParseSettings = %+v, want the configured token trimmed", got)
	}
}
