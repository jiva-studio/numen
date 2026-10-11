package recognition

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

func TestUnsetThreadsAndJobsTakeTheResolvedProfile(t *testing.T) {
	cfg := Defaults()
	cfg.Compute = Compute{Threads: 6, Jobs: 3}
	if got := cfg.threads(); got != 6 {
		t.Errorf("the recogniser takes %d threads, want 6", got)
	}
	if got := cfg.jobs(); got != 3 {
		t.Errorf("the recogniser runs %d jobs, want 3", got)
	}
}

func TestExplicitThreadsAndJobsWinOverTheResolvedProfile(t *testing.T) {
	cfg := Defaults()
	cfg.Compute = Compute{Threads: 6, Jobs: 3}
	cfg.Recognise.Threads, cfg.Recognise.Jobs = 2, 5
	if got := cfg.threads(); got != 2 {
		t.Errorf("the recogniser takes %d threads, want 2", got)
	}
	if got := cfg.jobs(); got != 5 {
		t.Errorf("the recogniser runs %d jobs, want 5", got)
	}
}

func TestAnUnresolvedProfileTakesTheFallbackCounts(t *testing.T) {
	cfg := Defaults()
	if got, want := cfg.threads(), min(fallbackThreads, onnxruntime.DefaultThreads()); got != want {
		t.Errorf("the recogniser takes %d threads, want %d", got, want)
	}
	if got := cfg.jobs(); got != fallbackJobs {
		t.Errorf("the recogniser runs %d jobs, want %d", got, fallbackJobs)
	}
}
