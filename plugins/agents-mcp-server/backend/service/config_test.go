package service

import "testing"

func TestReadConfigReadsTheTokenValueOnly(t *testing.T) {
	t.Setenv("MCP_TOKEN", "env-token-5150")

	if cfg := readConfig(nil); cfg.Token != "" {
		t.Errorf("readConfig without values = %+v, want an empty token", cfg)
	}
	if cfg := readConfig(map[string]string{"port": "9002", "token": " map-token "}); cfg.Token != "map-token" {
		t.Errorf("readConfig = %+v, want the configured token trimmed", cfg)
	}
}
