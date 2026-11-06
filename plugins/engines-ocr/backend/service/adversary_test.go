package service

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
)

// A recognition the palette command runs is a recognition of the file in progress. A tool call for
// the same file must not start a second one.
func TestRecognitionCreateOfAScanThePaletteCommandRecognisesReportsItRunning(t *testing.T) {
	s := newToolScene(t)
	done := make(chan error, 1)
	go func() {
		_, err := newRecogniseCommand(s.tools.recognitions)(t.Context(), command.Request{Command: "recognition.create", Path: scanPath})
		done <- err
	}()
	s.waitEntered(t)

	out := s.recognise(t, "Physics", scanPath)

	if out.State != stateRunning {
		t.Errorf("recognition_create while the command recognises the scan returned %+v, want the state running", out)
	}
	close(s.recogniser.release)
	if err := <-done; err != nil {
		t.Errorf("the command: %v", err)
	}
	s.waitFinished(t, out.TaskID)
}

// The palette command run while the tool's recognition holds the scan is refused as already being
// recognised.
func TestRecogniseCommandOfAScanTheToolRecognisesFails(t *testing.T) {
	s := newToolScene(t)
	out := s.recognise(t, "Physics", scanPath)
	s.waitEntered(t)

	_, err := newRecogniseCommand(s.tools.recognitions)(t.Context(), command.Request{Command: "recognition.create", Path: scanPath})

	if !errors.Is(err, recognition.ErrAlreadyRecognising) {
		t.Errorf("the command while the tool recognises the scan returned %v, want it already being recognised", err)
	}
	close(s.recogniser.release)
	s.waitFinished(t, out.TaskID)
}
