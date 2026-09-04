package container_test

import (
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/container"
)

// How many cores this machine has is read here and nowhere else, so the
// simulator behind a preset's control is told it rather than asking.
func TestTheCurvesAssembledAreToldTheMachinesCores(t *testing.T) {
	held := container.Config{ServiceDir: ".numen"}.Flashcards(nil, nil, nil)
	if got := held.Curves.Cores; got != runtime.GOMAXPROCS(0) {
		t.Errorf("the curves run %d places at once, and the machine runs %d", got, runtime.GOMAXPROCS(0))
	}
}
