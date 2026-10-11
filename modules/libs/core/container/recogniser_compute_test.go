package container

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
)

func TestTheRecogniserIsGivenTheThreadsAndSessionsOfTheProfile(t *testing.T) {
	cfg := Config{ResolvedCompute: settings.ResolvedCompute{OcrThreads: 8, OcrSessions: 2}}
	got := cfg.getRecognition().Compute
	if got.Threads != 8 || got.Jobs != 2 {
		t.Errorf("the recogniser is given %+v, want 8 threads and 2 jobs", got)
	}
}

func TestAContainerWithoutAProfileLeavesTheRecogniserCountsUnresolved(t *testing.T) {
	got := Config{}.getRecognition().Compute
	if got.Threads != 0 || got.Jobs != 0 {
		t.Errorf("the recogniser is given %+v, want none", got)
	}
}
