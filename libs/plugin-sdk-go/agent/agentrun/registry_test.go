package agentrun_test

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cleanup"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
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

	w := agentrun.NewWork(conversation, 1, agentrun.StopByCancel(cancel))
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
		cleanup.Wait(t, cmd)
		close(doneCh)
	}()
	t.Cleanup(func() {
		if err := w.Stop(); err != nil {
			t.Error(err)
		}
	})
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

// Close returns only after every child has finished, and rejects work after it.
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
	if registry.Add(agentrun.NewWork("", 1, func() error { return nil })) {
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

// Close waits for work added to the registry until its reader reports it is
// done, even when the reader has not started.
func TestCloseWaitsForTheReaderOfWhatWasAdded(t *testing.T) {
	var registry agentrun.Registry
	w := agentrun.NewWork("", 1, func() error { return nil })
	if !registry.Add(w) {
		t.Fatal("the registry rejected work while open")
	}

	closed := make(chan struct{})
	go func() {
		if err := registry.Close(); err != nil {
			t.Error(err)
		}
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

	res, err := registry.FinishConversation(t.Context(), connect.NewRequest(&v1.FinishConversationRequest{Conversation: "c-1"}))
	if err != nil || res == nil {
		t.Fatalf("got %v, %v", res, err)
	}
	if got := registry.GetSession("c-1"); got != "" {
		t.Errorf("the session is %q, want none", got)
	}
}

// Close returns the error of a stop function that fails, after the reader has ended.
func TestCloseReturnsTheErrorOfAStopFunction(t *testing.T) {
	var registry agentrun.Registry
	stopErr := errors.New("kill failed")
	w := agentrun.NewWork("", 1, func() error { return stopErr })
	if !registry.Add(w) {
		t.Fatal("the registry rejected work while open")
	}
	w.End()

	if err := registry.Close(); !errors.Is(err, stopErr) {
		t.Fatalf("Close = %v, want %v", err, stopErr)
	}
}
