package service

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/usecase/feed"
)

func (w *wire) deliver(t *testing.T, event string) {
	t.Helper()
	if err := w.auto.HandleEvent(t.Context(), event); err != nil {
		t.Fatal(err)
	}
}

func TestAFilesChangedEventTranscribesTheRecordingOfTheFeedOnceInTheBackground(t *testing.T) {
	w := newWire(t)
	w.auto.UpdateSettings(map[string]string{feed.SettingAutomatic: "true"})

	w.deliver(t, "onFilesChanged:Lectures")
	w.waitUntilStarted(t)
	for _, task := range w.queue.ListTasks() {
		if task.IsUserStarted {
			t.Errorf("the task %s is shown as started by a person", task.ID)
		}
	}
	w.deliver(t, "onFilesChanged:Lectures")
	close(w.recognizer.release)
	w.waitUntilNoTask(t)
	w.deliver(t, "onFilesChanged:Lectures")

	listing, err := w.texts.ListVersions(t.Context(), lecture)
	if err != nil || listing.Live == "" {
		t.Errorf("the recording lists %+v (%v), want its transcript live", listing, err)
	}
	if opened := w.recognizer.opened.Load(); opened != 1 {
		t.Errorf("the recording was opened %d times, want once", opened)
	}
}

func TestAFilesChangedEventTranscribesNothingWhileTheSettingIsOff(t *testing.T) {
	w := newWire(t)

	w.deliver(t, "onFilesChanged:Lectures")
	w.deliver(t, "onIndexChanged:Lectures")
	w.transcriber.Close()

	if opened := w.recognizer.opened.Load(); opened != 0 {
		t.Errorf("the recording was opened %d times, want never", opened)
	}
}

func TestTurningTheSettingOnTranscribesTheRecordingsOfTheShownVault(t *testing.T) {
	w := newWire(t)

	w.auto.UpdateSettings(map[string]string{feed.SettingAutomatic: "true"})
	w.waitUntilStarted(t)
	close(w.recognizer.release)
	w.waitUntilNoTask(t)

	listing, err := w.texts.ListVersions(t.Context(), lecture)
	if err != nil || listing.Live == "" {
		t.Errorf("the recording lists %+v (%v), want its transcript live", listing, err)
	}
}

func TestAResetOfTheFeedSkipsARecordingWithItsTranscript(t *testing.T) {
	w := newWire(t)
	w.auto.UpdateSettings(map[string]string{feed.SettingAutomatic: "true"})
	w.deliver(t, "onFilesChanged:Lectures")
	w.waitUntilStarted(t)
	close(w.recognizer.release)
	w.waitUntilNoTask(t)

	w.vault.ResetFeed()
	w.deliver(t, "onFilesChanged:Lectures")
	w.transcriber.Close()

	if opened := w.recognizer.opened.Load(); opened != 1 {
		t.Errorf("the recording was opened %d times, want once", opened)
	}
}
