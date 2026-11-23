package logs_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
)

func TestParseAndResolveLevel(t *testing.T) {
	cases := map[string]struct {
		input string
		want  logs.Level
		ok    bool
	}{
		"debug":   {"debug", logs.LevelDebug, true},
		"info":    {"Info", logs.LevelInfo, true},
		"warning": {"warning", logs.LevelWarning, true},
		"warn":    {"warn", logs.LevelWarning, true},
		"error":   {"ERROR", logs.LevelError, true},
		"unknown": {"trace", logs.DefaultLevel, false},
		"empty":   {"", logs.DefaultLevel, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, ok := logs.ParseLevel(tc.input)
			if ok != tc.ok {
				t.Fatalf("ParseLevel(%q) ok = %v, want %v", tc.input, ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Errorf("ParseLevel(%q) = %v, want %v", tc.input, got, tc.want)
			}
			if resolved := logs.ResolveLevel(tc.input); resolved != tc.want {
				t.Errorf("ResolveLevel(%q) = %v, want %v", tc.input, resolved, tc.want)
			}
		})
	}
}

func TestLevelAllowsHigherOrEqualSeverity(t *testing.T) {
	if !logs.LevelInfo.CanRecord(logs.LevelInfo) || !logs.LevelInfo.CanRecord(logs.LevelWarning) || !logs.LevelInfo.CanRecord(logs.LevelError) {
		t.Error("LevelInfo should allow info, warning and error")
	}
	if logs.LevelInfo.CanRecord(logs.LevelDebug) {
		t.Error("LevelInfo should not allow debug")
	}
	if !logs.LevelDebug.CanRecord(logs.LevelDebug) {
		t.Error("LevelDebug should allow debug")
	}
	if logs.LevelError.CanRecord(logs.LevelWarning) {
		t.Error("LevelError should not allow warning")
	}
}

func TestNormalizeRedactsSecretsAndTruncatesOversizedValues(t *testing.T) {
	instant := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	longVal := `"` + strings.Repeat("x", logs.MaxPropertyBytes+100) + `"`

	entry, err := logs.Normalize(logs.Entry{
		Time:    instant,
		Message: "  NoteService/CreateNote  ",
		Properties: map[string]string{
			"apiKey":     `"sk-secret"`,
			"authToken":  `"bearer-123"`,
			"durationMs": "4.2",
			"payload":    longVal,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Source != logs.SourceCore || entry.Level != logs.LevelInfo || entry.Message != "NoteService/CreateNote" {
		t.Errorf("entry = %+v", entry)
	}
	if entry.Properties["apiKey"] != logs.RedactedValue || entry.Properties["authToken"] != logs.RedactedValue {
		t.Errorf("secrets were not redacted: %+v", entry.Properties)
	}
	if entry.Properties["durationMs"] != "4.2" {
		t.Errorf("durationMs = %q, want 4.2", entry.Properties["durationMs"])
	}
	if !strings.Contains(entry.Properties["payload"], "[truncated]") {
		t.Errorf("oversized property was not truncated: len=%d", len(entry.Properties["payload"]))
	}
}

func TestNormalizeRefusesEmptyMessage(t *testing.T) {
	_, err := logs.Normalize(logs.Entry{Message: "   "})
	if !errors.Is(err, logs.ErrEmptyMessage) {
		t.Errorf("err = %v, want %v", err, logs.ErrEmptyMessage)
	}
}
