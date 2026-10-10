package acp

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func TestPlaces_FindBinaries(t *testing.T) {
	agy := FindAntigravity()
	if agy == "" {
		t.Errorf("expected non-empty antigravity command")
	}

	codex := FindCodex()
	if codex == "" {
		t.Errorf("expected non-empty codex command")
	}
}

func TestAgent_FinishEmpty(t *testing.T) {
	agent := &Agent{Program: "antigravity"}
	if err := agent.Finish(t.Context(), ""); err != nil {
		t.Errorf("unexpected error on empty finish: %v", err)
	}
}

func TestAgent_TakeWithoutTools(t *testing.T) {
	agent := &Agent{Program: "antigravity"}
	_, err := agent.Take(t.Context(), port.Task{Question: "hello"})
	if err == nil {
		t.Errorf("expected error when no tools provided")
	}
}
