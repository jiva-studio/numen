package main

import (
	"context"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}

	if reg.works == nil {
		t.Error("expected non-nil works map")
	}
	if reg.sessions == nil {
		t.Error("expected non-nil sessions map")
	}

	reg.SetSession("conv-1", "sess-1")
	if sess := reg.GetSession("conv-1"); sess != "sess-1" {
		t.Errorf("expected sess-1, got %q", sess)
	}

	work := &Work{conversation: "conv-1"}
	if !reg.Add(work) {
		t.Error("expected add to succeed")
	}

	reg.Remove(work)

	if err := reg.EndConversation(context.Background(), "conv-1"); err != nil {
		t.Fatalf("unexpected error ending conversation: %v", err)
	}
	if sess := reg.GetSession("conv-1"); sess != "" {
		t.Errorf("expected empty session after end, got %q", sess)
	}

	if err := reg.Close(); err != nil {
		t.Fatalf("unexpected error closing registry: %v", err)
	}
	if reg.Add(work) {
		t.Error("expected add after close to return false")
	}
}
