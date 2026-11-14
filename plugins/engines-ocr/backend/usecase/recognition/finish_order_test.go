package recognition_test

import (
	"context"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/document"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/ports"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/recognition"
)

// holdingFinishQueue holds FinishTask until the test releases it, after it has told the test that
// the task is ending.
type holdingFinishQueue struct {
	host.Queue
	ending chan struct{}
	let    chan struct{}
	once   *sync.Once
}

func (q holdingFinishQueue) FinishTask(ctx context.Context, id string) error {
	q.once.Do(func() { close(q.ending) })
	<-q.let
	return q.Queue.FinishTask(ctx, id)
}

func TestIsRunningIsFalseWhenTheTaskEnds(t *testing.T) {
	s := newScene(t)
	queue := holdingFinishQueue{Queue: s.queue, ending: make(chan struct{}), let: make(chan struct{}), once: &sync.Once{}}
	letGo := sync.OnceFunc(func() { close(queue.let) })
	t.Cleanup(letGo)
	run, err := recognition.New(
		recognition.Host{Vaults: func(id domain.VaultID) recognition.Files { return s.vault.With(id) }, Store: foldersOf(s.store), Texts: s.textsOf, Queue: queue, Documents: document.NewReader()},
		func(context.Context) (ports.Recognizer, error) { return s.recognizer, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	runEnd, isStarted, err := run.Start(t.Context(), recognition.Request{Vault: "Physics", Path: "books/scan.pdf"})
	if err != nil || !isStarted {
		t.Fatalf("Start = %v, %v", isStarted, err)
	}
	<-queue.ending

	isRunning, err := run.IsRunning(t.Context(), "Physics", "books/scan.pdf")

	if err != nil || isRunning {
		t.Errorf("IsRunning = %v, %v while the task ends, want false", isRunning, err)
	}
	letGo()
	<-runEnd
}
