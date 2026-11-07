package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
)

// The palette command transcribes in the vault the host shows, passed as no vault. A tool call that
// names that vault by its identity is the same recording, and must not start a second run.
func TestTranscriptCreateDuringPaletteTranscriptionReportsRunning(t *testing.T) {
	s := newToolFixture(t)
	result := make(chan error, 1)
	go func() {
		_, err := newTranscribeCommand(s.transcriber)(t.Context(), command.Request{Command: "transcript.create", Path: lecture})
		result <- err
	}()
	s.waitStarted(t)

	out := s.createTranscript(t, "Lectures", lecture)

	if out.State != stateRunning {
		t.Errorf("transcript_create in the shown vault while the command transcribes the recording returned %+v, want the state running", out)
	}
	close(s.blocking.release)
	if err := <-result; err != nil {
		t.Errorf("the command: %v", err)
	}
}
