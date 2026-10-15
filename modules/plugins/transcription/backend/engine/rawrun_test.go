package engine

import (
	"errors"
	"slices"
	"testing"
	"unsafe"
)

// fakeRuntime records every call a rawRun makes and answers with the outputs it
// is given.
type fakeRuntime struct {
	calls    []string
	released []uintptr
	status   uintptr
	output   []float32
	run      *rawRun
}

func (f *fakeRuntime) get() rawCalls {
	return rawCalls{
		run: func(_, _, _, _, _, _, _, _ uintptr) uintptr {
			f.calls = append(f.calls, "run")
			for i := range f.run.outputs {
				if f.run.outputs[i] == 0 {
					f.run.outputs[i] = uintptr(100 + i)
				}
			}
			return f.status
		},
		getData: func(_, _ uintptr) uintptr {
			f.calls = append(f.calls, "data")
			f.run.data = unsafe.Pointer(&f.output[0])
			return 0
		},
		release: func(value uintptr) {
			f.calls = append(f.calls, "release")
			f.released = append(f.released, value)
		},
		check: func(status uintptr) error {
			if status != 0 {
				return errors.New("the run failed")
			}
			return nil
		},
	}
}

func newFakeRun(f *fakeRuntime, outputs int) *rawRun {
	names := make([][]byte, outputs)
	for i := range names {
		names[i] = []byte("out\x00")
	}
	f.run = newRawRunOf(f.get(), 7, [][]byte{[]byte("in\x00")}, names, []uintptr{1})
	return f.run
}

func TestARawRunKeepsItsOutputsAcrossRunsAndReleasesEachOnce(t *testing.T) {
	f := &fakeRuntime{output: []float32{1, 2, 3}}
	run := newFakeRun(f, 2)

	for range 3 {
		if err := run.run(); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.released) != 0 {
		t.Fatalf("outputs were released while the run was in use: %v", f.released)
	}
	run.close()
	run.close()

	if want := []uintptr{100, 101}; !slices.Equal(f.released, want) {
		t.Errorf("released %v, want %v once each", f.released, want)
	}
}

func TestARawRunCopiesAnOutputOutBeforeTheNextRun(t *testing.T) {
	f := &fakeRuntime{output: []float32{1, 2, 3}}
	run := newFakeRun(f, 1)
	if err := run.run(); err != nil {
		t.Fatal(err)
	}

	got := make([]float32, 2)
	if err := run.copyOutput(0, got); err != nil {
		t.Fatal(err)
	}
	f.output[0] = 9
	if got[0] != 1 || got[1] != 2 {
		t.Errorf("the copy followed the output: %v", got)
	}
	if want := []string{"run", "data"}; !slices.Equal(f.calls, want) {
		t.Errorf("calls %v, want %v", f.calls, want)
	}
}

func TestAFailedRawRunStillReleasesWhatItCreated(t *testing.T) {
	f := &fakeRuntime{status: 1, output: []float32{1}}
	run := newFakeRun(f, 3)

	if err := run.run(); err == nil {
		t.Fatal("a failed run was taken for a good one")
	}
	run.close()

	if len(f.released) != 3 {
		t.Errorf("released %v after a failed run, want the 3 outputs", f.released)
	}
}
