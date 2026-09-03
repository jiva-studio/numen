package container

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/task"
)

// The model's own step draws no share until some of the model is here. A count
// of nothing against a size is a row sitting at nought per cent for the length
// of a download, which says less than a ring that turns.
func TestTheModelDrawsNoShareBeforeAnyOfItIsHere(t *testing.T) {
	tasks := task.New()
	tell := preparing(tasks, listing{id: "getting ready: indexing: a/model", name: "a/model"})

	// Nothing is known yet: the step is in the list from the moment it starts.
	tell(0, 0)
	at := only(t, tasks)
	if at.Doing != "Preparing the model" || at.About != "a/model" {
		t.Errorf("the step is shown as %q about %q", at.Doing, at.About)
	}
	if at.Total != 0 {
		t.Errorf("a step that has counted nothing is drawn against %d", at.Total)
	}

	// The cache is read before the first bytes land, and the size is known.
	tell(0, 90_000_000)
	if at := only(t, tasks); at.Total != 0 || at.Done != 0 {
		t.Errorf("a model with none of it here is drawn as %d of %d", at.Done, at.Total)
	}

	tell(30_000_000, 90_000_000)
	at = only(t, tasks)
	if at.Done != 30_000_000 || at.Total != 90_000_000 {
		t.Errorf("the model is drawn as %d of %d", at.Done, at.Total)
	}
	if at.Counting != task.Bytes {
		t.Errorf("a model is counted as %v", at.Counting)
	}
}

// only is the one piece of work in the list.
func only(t *testing.T, tasks *task.Tasks) task.Task {
	t.Helper()
	held := tasks.List()
	if len(held) != 1 {
		t.Fatalf("the list holds %d pieces of work: %+v", len(held), held)
	}
	return held[0]
}
