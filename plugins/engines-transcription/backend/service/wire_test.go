package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/hostclient"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/usecase/transcribe"
)

const (
	lecture      = "assets/lecture.wav"
	lectureSound = "RIFF the bytes of a lecture"
	// workDeadline is how long a transcription may run before the test fails.
	workDeadline = time.Minute
)

// listeningRecognizer returns one stretch of a recording that reads "first words", but opening a
// recording waits until release is closed. The first recording it opens closes startCh.
type listeningRecognizer struct {
	startCh   chan struct{}
	startOnce sync.Once
	release   chan struct{}
}

func (r *listeningRecognizer) Open(ctx context.Context, _ []byte) (transcribe.Recording, error) {
	r.startOnce.Do(func() { close(r.startCh) })
	select {
	case <-r.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return oneStretch{}, nil
}

func (*listeningRecognizer) Transcribe(context.Context, transcribe.Audio) (string, error) {
	return "first words", nil
}

type oneStretch struct{}

func (oneStretch) GetDuration() int { return 1000 }

func (oneStretch) GetSegments(_ context.Context, from, _ int) ([]transcribe.Audio, error) {
	if from >= 1000 {
		return nil, nil
	}
	return []transcribe.Audio{{Samples: nil, From: 0, To: 1000}}, nil
}

func (oneStretch) Close() error { return nil }

// wire is the commands and the tool of the plugin over the real transcriber, the vault Lectures of
// the test host and a recognizer that waits for the test.
type wire struct {
	vault       *hosttest.Vault
	queue       *hosttest.Queue
	texts       *hosttest.Texts
	recognizer  *listeningRecognizer
	transcriber *transcribe.Transcriber
	registry    *tools.Registry
	commands    *command.Registry
}

func newWire(t *testing.T) *wire {
	t.Helper()
	vault := hosttest.New("Lectures")
	vault.File(lecture, lectureSound)
	store := vault.Store()
	texts := hosttest.NewTexts(vault.Vault().ID, transcribe.PluginID)
	texts.File(lecture, []byte(lectureSound))
	texts.SetStore(transcribe.PluginID, store)
	w := &wire{
		vault: vault, queue: hosttest.NewQueue(), texts: texts,
		recognizer: &listeningRecognizer{startCh: make(chan struct{}), startOnce: sync.Once{}, release: make(chan struct{})},
	}
	vaults := func(id domain.VaultID) hostclient.VaultReader { return vault.With(id) }
	w.transcriber = transcribe.NewTranscriber(hostclient.NewFiles(vaults),
		func(id domain.VaultID) transcribe.Folder { return store.With(id) },
		func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		w.queue,
		func(context.Context) (transcribe.Recognizer, error) { return w.recognizer, nil }, func(string, error) {})
	t.Cleanup(w.transcriber.Close)
	requests := transcribe.NewRequests(w.transcriber, func(id domain.VaultID) transcribe.Vault { return vault.With(id) })
	registry, err := newTools(requests).registry()
	if err != nil {
		t.Fatal(err)
	}
	commands, err := newCommands(w.transcriber)
	if err != nil {
		t.Fatal(err)
	}
	w.registry, w.commands = registry, commands
	return w
}

func (w *wire) waitUntilStarted(t *testing.T) {
	t.Helper()
	select {
	case <-w.recognizer.startCh:
	case <-time.After(workDeadline):
		t.Fatalf("no recording was opened within %v; tasks %+v", workDeadline, w.queue.ListTasks())
	}
}

func (w *wire) waitUntilNoTask(t *testing.T) {
	t.Helper()
	deadline := time.After(workDeadline)
	for len(w.queue.ListTasks()) != 0 {
		select {
		case <-deadline:
			t.Fatalf("the tasks %+v did not end within %v", w.queue.ListTasks(), workDeadline)
		case <-time.After(time.Millisecond):
		}
	}
}
