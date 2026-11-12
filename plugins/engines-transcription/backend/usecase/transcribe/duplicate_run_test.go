package transcribe

import "testing"

// The palette command transcribes in the vault the host shows, passed as no vault. A request that
// names that vault by its identity is the same recording, and must not start a second run.
func TestTranscriptCreateDuringPaletteTranscriptionReportsRunning(t *testing.T) {
	s := newRequestFixture(t)
	result := make(chan error, 1)
	go func() {
		_, err := s.transcriber.Start(t.Context(), "", lecture)
		result <- err
	}()
	s.waitUntilStarted(t)

	out := s.createTranscript(t, "Lectures", lecture)

	if out.State != stateRunning || out.TaskID != getTaskID("Lectures", lecture) {
		t.Errorf("a request in the shown vault while the command transcribes the recording returned %+v, want the state running", out)
	}
	close(s.blocking.release)
	if err := <-result; err != nil {
		t.Errorf("the command: %v", err)
	}
}
