package container

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/run"
)

// A failed read of the embedding model is listed as a task that carries the failure as its reason.
func TestAFailedReadOfTheEmbeddingModelIsListedAsATaskWithItsReason(t *testing.T) {
	tasks := run.NewTasks()

	newModelReadReporter(tasks)(errors.New("read model of embedding plugin md.numen.embeddings: unavailable"))

	listed := tasks.List()
	if len(listed) != 1 {
		t.Fatalf("%d tasks, want 1", len(listed))
	}
	if listed[0].Error != "read model of embedding plugin md.numen.embeddings: unavailable" {
		t.Errorf("the task reason is %q", listed[0].Error)
	}
}

// Without tasks there is nowhere to report to, and no reporter.
func TestNoReporterOfAFailedModelReadWithoutTasks(t *testing.T) {
	if newModelReadReporter(nil) != nil {
		t.Error("got a reporter without tasks")
	}
}
