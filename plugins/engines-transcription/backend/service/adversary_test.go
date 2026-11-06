package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
)

// The palette command transcribes in the vault the host shows, passed as no vault. A tool call that
// names that vault by its identity is the same recording, and must not start a second run.
func TestTranscriptCreateOfARecordingThePaletteCommandTranscribesReportsItRunning(t *testing.T) {
	s := newToolScene(t)
	done := make(chan error, 1)
	go func() {
		_, err := newTranscribeCommand(s.transcriber)(t.Context(), command.Request{Command: "transcript.create", Path: lecture})
		done <- err
	}()
	s.waitEntered(t)

	out := s.createTranscript(t, "Lectures", lecture)

	if out.State != stateRunning {
		t.Errorf("transcript_create in the shown vault while the command transcribes the recording returned %+v, want the state running", out)
	}
	close(s.blocking.release)
	if err := <-done; err != nil {
		t.Errorf("the command: %v", err)
	}
}
