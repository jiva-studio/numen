package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
)

func (w *wire) runUpdate(t *testing.T) {
	t.Helper()
	if _, err := w.commands.RunCommand(t.Context(), command.Request{Command: UpdateCommand, Path: lecture}); err != nil {
		t.Fatal(err)
	}
}

func TestTheUpdateCommandTranscribesTheRecordingOfTheFeedOnce(t *testing.T) {
	w := newWire(t)

	w.runUpdate(t)
	w.waitUntilStarted(t)
	w.runUpdate(t)
	close(w.recognizer.release)
	w.waitUntilNoTask(t)
	w.runUpdate(t)

	listing, err := w.texts.ListVersions(t.Context(), lecture)
	if err != nil || listing.Live == "" {
		t.Errorf("the recording lists %+v (%v), want its transcript live", listing, err)
	}
	if opened := w.recognizer.opened.Load(); opened != 1 {
		t.Errorf("the recording was opened %d times, want once", opened)
	}
}

func TestAResetOfTheFeedSkipsARecordingWithItsTranscript(t *testing.T) {
	w := newWire(t)
	w.runUpdate(t)
	w.waitUntilStarted(t)
	close(w.recognizer.release)
	w.waitUntilNoTask(t)

	w.vault.ResetFeed()
	w.runUpdate(t)
	w.transcriber.Close()

	if opened := w.recognizer.opened.Load(); opened != 1 {
		t.Errorf("the recording was opened %d times, want once", opened)
	}
}
