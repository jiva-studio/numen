package service

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/usecase/transcribe"
)

func TestTheHandlerOfCreateTextAnswersWhileTheTaskTranscribes(t *testing.T) {
	w := newWire(t)

	res, err := w.commands.RunCommand(t.Context(), command.Request{Command: "app:file.createText", Path: lecture})

	if err != nil || res.Message != "transcribing "+lecture {
		t.Fatalf("answered %+v (%v)", res, err)
	}
	w.waitUntilStarted(t)
	close(w.recognizer.release)
	w.waitUntilNoTask(t)
	listing, err := w.texts.ListVersions(t.Context(), lecture)
	if err != nil || listing.Live == "" {
		t.Errorf("the recording lists %+v (%v) once the task ended, want the transcript live", listing, err)
	}
}

func TestTheHandlerOfCreateTextFailsForARecordingBeingTranscribed(t *testing.T) {
	w := newWire(t)
	if _, err := w.commands.RunCommand(t.Context(), command.Request{Command: "app:file.createText", Path: lecture}); err != nil {
		t.Fatal(err)
	}
	w.waitUntilStarted(t)

	_, err := w.commands.RunCommand(t.Context(), command.Request{Command: "app:file.createText", Path: lecture})

	if !errors.Is(err, transcribe.ErrAlreadyTranscribing) {
		t.Errorf("a second run while the first transcribes answered %v, want ErrAlreadyTranscribing", err)
	}
	close(w.recognizer.release)
	w.waitUntilNoTask(t)
}

func TestThePluginRunsNeitherDeleteTextNorTheCommandOfBefore(t *testing.T) {
	w := newWire(t)

	if _, err := w.commands.RunCommand(t.Context(), command.Request{Command: "app:version.delete", Path: lecture}); !errors.Is(err, command.ErrUnknownCommand) {
		t.Errorf("Delete text, which the core runs, answered %v, want ErrUnknownCommand", err)
	}
	if _, err := w.commands.RunCommand(t.Context(), command.Request{Command: "transcript.create", Path: lecture}); !errors.Is(err, command.ErrUnknownCommand) {
		t.Errorf("the command of before answered %v, want ErrUnknownCommand", err)
	}
}
