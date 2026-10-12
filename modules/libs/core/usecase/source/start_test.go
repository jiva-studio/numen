package source

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func TestRecognitionIsStartedAndBeginsWhereNothingIsGoing(t *testing.T) {
	w := newWatched(t, errors.New("nothing to read with"))
	if got := NewStartRecognition(w.RecognitionWorker).Execute(somewhere, "a.pdf"); got != port.Began {
		t.Errorf("the first document started as %v", got)
	}
	w.waitUntilDone(t)
}

func TestRecognitionStatusSaysWhetherTheModelsAreHere(t *testing.T) {
	w := newWatched(t, nil)
	status := NewGetRecognitionStatus(w.RecognitionWorker)
	if got := status.Execute(); !got.IsReady || got.IsRunning {
		t.Errorf("an idle worker holding the models reported %+v", got)
	}
	w.RecognitionWorker.with.Runtime.Ready = func() bool { return false }
	if got := status.Execute(); got.IsReady {
		t.Errorf("a worker holding no models reported %+v", got)
	}
}

func TestTranscriptionReadinessFollowsTheModels(t *testing.T) {
	listening, _ := newTranscriptionWorker(t, &deaf{})
	readiness := NewGetTranscriptionReadiness(listening)
	if !readiness.Execute() {
		t.Error("a worker holding the models was not ready")
	}
	listening.with.Runtime.Ready = func() bool { return false }
	if readiness.Execute() {
		t.Error("a worker holding no models was ready")
	}
}
