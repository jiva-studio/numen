package service

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/launch"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
)

// A recognition the palette command runs is a recognition of the file in progress. A tool call for
// the same file must not start a second one.
func TestRecognitionCreateDuringPaletteRecognitionReportsRunning(t *testing.T) {
	s := newToolFixture(t)
	result := make(chan error, 1)
	go func() {
		_, err := newRecogniseCommand(s.launcher)(t.Context(), command.Request{Command: "app:file.createText", Path: scanPath})
		result <- err
	}()
	s.waitStarted(t)

	out := s.recognize(t, "Physics", scanPath)

	if out.State != launch.StateRunning {
		t.Errorf("recognition_create while the command recognizes the scan returned %+v, want the state running", out)
	}
	close(s.recogniser.release)
	if err := <-result; err != nil {
		t.Errorf("the command: %v", err)
	}
	s.waitFinished(t, out.TaskID)
}

// The palette command run while the tool's recognition has the scan is rejected as already being
// recognized.
func TestRecogniseCommandDuringToolRecognitionFails(t *testing.T) {
	s := newToolFixture(t)
	out := s.recognize(t, "Physics", scanPath)
	s.waitStarted(t)

	_, err := newRecogniseCommand(s.launcher)(t.Context(), command.Request{Command: "app:file.createText", Path: scanPath})

	if !errors.Is(err, recognition.ErrAlreadyRecognising) {
		t.Errorf("the command while the tool recognizes the scan returned %v, want it already being recognized", err)
	}
	close(s.recogniser.release)
	s.waitFinished(t, out.TaskID)
}
