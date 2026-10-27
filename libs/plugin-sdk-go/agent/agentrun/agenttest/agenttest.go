// Package agenttest holds the helpers the tests of agent plugins share: shell scripts that stand
// in for the program a plugin runs, and a client of the plugin's agent service.
package agenttest

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
)

// NeedsShell skips a test that runs a script where the system has no shell to run it.
func NeedsShell(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fixtures are shell scripts")
	}
}

// WriteScript writes an executable shell script and returns its path.
func WriteScript(t *testing.T, name, body string) string {
	t.Helper()
	NeedsShell(t)
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// QuoteShell wraps a string in single quotes, escaping the single quotes inside it, for a shell
// script.
func QuoteShell(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Serve serves the handler over HTTP for the length of the test and returns a client of it.
func Serve(t *testing.T, handler agentrun.AgentServiceHandler) agentrun.AgentServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(agentrun.NewAgentServiceHandler(handler))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return agentrun.NewAgentServiceClient(http.DefaultClient, server.URL)
}

// RecordingProgram writes a program that records the folder it ran in and its arguments, one on a
// line, and returns its path with the function that reads the record.
func RecordingProgram(t *testing.T, name string) (string, func() (dir string, argv []string)) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "run")
	path := WriteScript(t, name,
		"pwd > "+QuoteShell(out+".dir")+"\n"+
			"for a in \"$@\"; do printf '%s\\n' \"$a\"; done > "+QuoteShell(out+".argv")+"\n")
	return path, func() (string, []string) {
		argv, err := os.ReadFile(out + ".argv")
		if err != nil {
			t.Fatal(err)
		}
		dir, err := os.ReadFile(out + ".dir")
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(dir)), strings.Split(strings.TrimSpace(string(argv)), "\n")
	}
}

// Ask sends the request and waits until the stream ends.
func Ask(t *testing.T, client agentrun.AgentServiceClient, req *agentrun.AskAgentRequest) {
	t.Helper()
	stream, err := client.AskAgent(t.Context(), connect.NewRequest(req))
	if err != nil {
		t.Fatal(err)
	}
	for stream.Receive() {
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
}

// IsDir reports whether got names the folder want, resolving links.
func IsDir(got, want string) bool {
	resolved, _ := filepath.EvalSymlinks(want)
	return got == want || got == resolved
}

// CheckLinkedHandler serves the handler a host that links the plugin in gets and checks that it
// answers GetAgentState with the token it was given and refuses a call without it.
func CheckLinkedHandler(t *testing.T, newHandler func(map[string]string) (http.Handler, error)) {
	t.Helper()
	handler, err := newHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	client := agentrun.NewAgentServiceClient(web.Client(), web.URL)

	req := connect.NewRequest(&agentrun.GetAgentStateRequest{})
	req.Header().Set("Authorization", "Bearer linked-token")
	if _, err := client.GetAgentState(t.Context(), req); err != nil {
		t.Fatalf("call with the token: %v", err)
	}
	_, err = client.GetAgentState(t.Context(), connect.NewRequest(&agentrun.GetAgentStateRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("call without the token = %v, want unauthenticated", err)
	}
}
