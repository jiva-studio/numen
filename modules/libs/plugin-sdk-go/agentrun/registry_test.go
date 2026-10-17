package agentrun_test

import (
	"context"
	"os/exec"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agentrun"
)

// start runs a child that waits until it is killed, and the reader that waits for it.
func start(t *testing.T, registry *agentrun.Registry, conversation string) (*agentrun.Work, <-chan struct{}) {
	t.Helper()

	running, cancel := context.WithCancel(t.Context())
	cmd := exec.CommandContext(running, "sh", "-c", "sleep 60")
	agentrun.Supervise(cmd)

	w := agentrun.NewWork(conversation, 1, cancel)
	if !registry.Add(w) {
		t.Fatal("the registry refused work while open")
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	over := make(chan struct{})
	w.Begin()
	go func() {
		defer w.End()
		defer registry.Remove(w)
		defer close(w.Output)
		_ = cmd.Wait()
		close(over)
	}()
	t.Cleanup(func() { _ = w.Stop() })
	return w, over
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
func TestCloseStopsEveryChildAndRefusesMore(t *testing.T) {
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
func TestNoConversationKeepsNoSession(t *testing.T) {
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
	w, over := start(t, &registry, "")

	if err := w.Stop(); err != nil {
		t.Fatal(err)
	}
	if !isOver(over) {
		t.Error("the child is still running after Stop")
	}
	if err := w.Stop(); err != nil {
		t.Fatal(err)
	}
}
