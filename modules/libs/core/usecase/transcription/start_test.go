package transcription

import "testing"

func TestTranscriptionReadinessFollowsTheModels(t *testing.T) {
	listening, _ := newTranscriptionWorker(t, &deaf{})
	readiness := NewGetReadiness(listening)
	if !readiness.Execute() {
		t.Error("a worker holding the models was not ready")
	}
	listening.with.Runtime.Ready = func() bool { return false }
	if readiness.Execute() {
		t.Error("a worker holding no models was ready")
	}
}
