package run_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
)

func TestAFailedReadOfTheEmbeddingModelIsListedAsATaskWithAFixedReason(t *testing.T) {
	tasks := run.NewTasks()

	run.NewModelReadReporter(tasks)(errors.New("read model of embedding plugin md.numen.embeddings: unavailable"))

	entries := tasks.List()
	if len(entries) != 1 {
		t.Fatalf("%d tasks, want 1", len(entries))
	}
	if entries[0].Error != "The embedding model could not be read. Check the embedding settings." {
		t.Errorf("the task reason is %q", entries[0].Error)
	}
}

func TestNoReporterOfAFailedModelReadWithoutTasks(t *testing.T) {
	if run.NewModelReadReporter(nil) != nil {
		t.Error("got a reporter without tasks")
	}
}
