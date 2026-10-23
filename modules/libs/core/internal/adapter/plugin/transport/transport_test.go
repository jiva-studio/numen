package transport

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestGenerateAuthToken(t *testing.T) {
	tok1, err := GenerateAuthToken()
	if err != nil {
		t.Fatalf("GenerateAuthToken failed: %v", err)
	}
	if len(tok1) != 64 {
		t.Fatalf("expected 64 hex characters, got length %d", len(tok1))
	}

	tok2, err := GenerateAuthToken()
	if err != nil {
		t.Fatalf("GenerateAuthToken failed: %v", err)
	}
	if tok1 == tok2 {
		t.Fatal("expected distinct random tokens, got identical")
	}
}

func TestParsePasseshake(t *testing.T) {
	tests := []struct {
		line     string
		wantPort int
		wantOk   bool
	}{
		{line: "NUMEN_READY port=8080", wantPort: 8080, wantOk: true},
		{line: "READY port=9999", wantPort: 9999, wantOk: true},
		{line: "info: server starting up...", wantPort: 0, wantOk: false},
		{line: "NUMEN_READY port=invalid", wantPort: 0, wantOk: false},
		{line: "NUMEN_READY other=123", wantPort: 0, wantOk: false},
	}

	for _, tt := range tests {
		gotPort, gotOk := ParseHandshake(tt.line)
		if gotOk != tt.wantOk || gotPort != tt.wantPort {
			t.Errorf("ParseHandshake(%q) = (%d, %v), want (%d, %v)",
				tt.line, gotPort, gotOk, tt.wantPort, tt.wantOk)
		}
	}
}

func TestWaitForPasseshakeSuccess(t *testing.T) {
	input := "initializing\nNUMEN_READY port=12345\nready to serve\n"
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	port, err := WaitForHandshake(ctx, strings.NewReader(input))
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if port != 12345 {
		t.Fatalf("expected port 12345, got %d", port)
	}
}

func TestWaitForPasseshakeFailure(t *testing.T) {
	input := "initializing\nfailed to start\n"
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	_, err := WaitForHandshake(ctx, strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
