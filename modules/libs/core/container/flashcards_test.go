package container_test

import (
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A machine naming no cache folder is what the schedules say they fall back
// for, so the store they answer with has to be one a caller can test against
// nil. A typed nil pointer inside the interface passes that test and panics on
// the first call.
func TestAMachineWithNoCacheFolderIsAnsweredNoStoreAtAll(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "")

	for _, one := range []struct {
		what  string
		asked func(container.Config) (port.ScheduleStore, error)
	}{
		{"the schedules", container.Config.Schedules},
		{"the counting", container.Config.OpenDayCounts},
	} {
		kept, err := one.asked(container.Config{ServiceDir: ".numen"})
		if err == nil {
			t.Errorf("%s found a folder on a machine that names none", one.what)
		}
		if kept != nil {
			t.Errorf("%s answered %#v, which is not nothing", one.what, kept)
		}
	}
}

// How many cores this machine has is read here and nowhere else, and the
// simulator behind a preset's control is told it.
func TestTheCurvesAssembledAreToldTheMachinesCores(t *testing.T) {
	held := container.Config{ServiceDir: ".numen"}.Flashcards(nil, nil, nil, nil)
	if got := held.Curves.Cores; got != runtime.GOMAXPROCS(0) {
		t.Errorf("the curves run %d places at once, and the machine runs %d", got, runtime.GOMAXPROCS(0))
	}
}
