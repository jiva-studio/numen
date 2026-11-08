package server

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestASecretLoggedWithTheStructuredLoggerShowsNoValue(t *testing.T) {
	one := Secret{ID: "s0000000a", Name: "Work", Group: "openrouter", Value: "invented-harbor-key"}
	var out bytes.Buffer
	slog.New(slog.NewJSONHandler(&out, nil)).Info("configure", "secret", one)

	if strings.Contains(out.String(), one.Value) {
		t.Errorf("the log holds the value:\n%s", out.String())
	}
	if !strings.Contains(out.String(), one.ID) {
		t.Errorf("the log lacks the id:\n%s", out.String())
	}
}
