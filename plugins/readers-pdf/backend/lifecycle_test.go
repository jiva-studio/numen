package main

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

func TestLifecycleConfigure(t *testing.T) {
	life := server.NewLifecycleServer(make(chan struct{}))

	success, err := life.ApplyConfig(t.Context(), map[string]string{
		"zoom": "1.5",
	})
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
	if !success {
		t.Errorf("Configure success = false, want true")
	}

	got, ok := life.GetConfig("zoom")
	if !ok || got != "1.5" {
		t.Errorf("zoom = %q, want %q", got, "1.5")
	}
}
