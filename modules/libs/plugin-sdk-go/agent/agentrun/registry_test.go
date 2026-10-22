package agentrun_test

import (
	"context"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// start runs a child that waits until it is killed, and the reader that waits for it.
func start(t *testing.T, registry *agentrun.Registry, conversation string) (*agentrun.Work, <-chan struct{}) {
	t.Helper()

	running, cancel := context.WithCancel(t.Context())
	command, args := "sh", []string{"-c", "sleep 60"}
	if runtime.GOOS == "windows" {
		command, args = "cmd", []string{"/c", "ping -n 60 127.0.0.1"}
	}
	cmd := exec.CommandContext(running, command, args...)
	agentrun.Supervise(cmd)

	w := agentrun.NewWork(conversation, 1, cancel)
	if !registry.Add(w) {
		t.Fatal("the registry rejected work while open")
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	doneCh := make(chan struct{})
	go func() {
		defer w.End()
		defer registry.Remove(w)
		defer close(w.Output)
		_ = cmd.Wait()
		close(doneCh)
	}()
	t.Cleanup(func() { _ = w.Stop() })
	return w, doneCh
}

// isOver is whether the child has been waited for.
func isOver(over <-chan struct{}) bool {
	select {
	case <-over:
		return true
	default:
		return false
	}
}

// Close returns only once every child is over, and refuses work after it.
func TestCloseStopsEveryChildAndRejectsMore(t *testing.T) {
	var registry agentrun.Registry
	_, first := start(t, &registry, "one")
	_, second := start(t, &registry, "")

	if err := registry.Close(); err != nil {
		t.Fatal(err)
	}
	if !isOver(first) || !isOver(second) {
		t.Errorf("a child outlived Close: first over %v, second over %v", isOver(first), isOver(second))
	}
	if registry.Add(agentrun.NewWork("", 1, func() {})) {
		t.Error("the registry took work after Close")
	}
}

// Ending a conversation stops what is answered in it and nothing else, and lets go of its session.
func TestEndConversationStopsOnlyThatConversation(t *testing.T) {
	var registry agentrun.Registry
	_, one := start(t, &registry, "one")
	_, two := start(t, &registry, "two")
	registry.SetSession("one", "s-one")
	registry.SetSession("two", "s-two")

	if err := registry.EndConversation(t.Context(), "one"); err != nil {
		t.Fatal(err)
	}
	if !isOver(one) {
		t.Error("the child of the ended conversation is still running")
	}
	if isOver(two) {
		t.Error("the child of another conversation was stopped")
	}
	if got := registry.GetSession("one"); got != "" {
		t.Errorf("session of the ended conversation is %q, want none", got)
	}
	if got := registry.GetSession("two"); got != "s-two" {
		t.Errorf("session of the other conversation is %q, want s-two", got)
	}
}

// A task that named no conversation keeps no session, and ending none is not an error.
func TestNoConversationPreservesNoSession(t *testing.T) {
	var registry agentrun.Registry
	registry.SetSession("", "s-lost")
	if got := registry.GetSession(""); got != "" {
		t.Errorf("session is %q, want none", got)
	}
	if err := registry.EndConversation(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
}

// Stop on one work ends its child and is safe to repeat.
func TestStopEndsTheChildOnce(t *testing.T) {
	var registry agentrun.Registry
	w, doneCh := start(t, &registry, "")

	if err := w.Stop(); err != nil {
		t.Fatal(err)
	}
	if !isOver(doneCh) {
		t.Error("the child is still running after Stop")
	}
	if err := w.Stop(); err != nil {
		t.Fatal(err)
	}
}

// A work the registry has taken in is waited for by Close until its reader says it is done, even
// where the reader has not started.
func TestCloseWaitsForTheReaderOfWhatWasAdded(t *testing.T) {
	var registry agentrun.Registry
	w := agentrun.NewWork("", 1, func() {})
	if !registry.Add(w) {
		t.Fatal("the registry rejected work while open")
	}

	closed := make(chan struct{})
	go func() {
		_ = registry.Close()
		close(closed)
	}()

	// Close stops the work before it waits for the reader, and cannot return before the reader ends.
	<-w.Stopped()
	select {
	case <-closed:
		t.Fatal("Close returned while the reader of the work was still to finish")
	default:
	}

	w.End()
	select {
	case <-closed:
	case <-time.After(10 * time.Second):
		t.Fatal("Close did not return once the reader finished")
	}
}

func TestFinishConversationEndsTheConversation(t *testing.T) {
	var registry agentrun.Registry
	registry.SetSession("c-1", "s-1")

	res, err := registry.FinishConversation(t.Context(), connect.NewRequest(&numenv1.FinishConversationRequest{Conversation: "c-1"}))
	if err != nil || res == nil {
		t.Fatalf("got %v, %v", res, err)
	}
	if got := registry.GetSession("c-1"); got != "" {
		t.Errorf("the session is %q, want none", got)
	}
}
