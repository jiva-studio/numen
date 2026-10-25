package recognition

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

func TestRecognitionIsStartedAndBeginsWhereNothingIsGoing(t *testing.T) {
	w := newWatched(t, errors.New("nothing to read with"))
	if got := NewStart(w.Worker).Execute(somewhere, "a.pdf"); got != windowport.Began {
		t.Errorf("the first document started as %v", got)
	}
	w.waitUntilDone(t)
}

func TestRecognitionStatusReturnsWhetherTheModelsAreHere(t *testing.T) {
	w := newWatched(t, nil)
	status := NewGetStatus(w.Worker)
	if got := status.Execute(); !got.IsReady || got.IsRunning {
		t.Errorf("an idle worker holding the models reported %+v", got)
	}
	w.Worker.with.Runtime.Ready = func() bool { return false }
	if got := status.Execute(); got.IsReady {
		t.Errorf("a worker holding no models reported %+v", got)
	}
}
