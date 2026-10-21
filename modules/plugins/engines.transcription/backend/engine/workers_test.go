package engine

import (
	"runtime"
	"slices"
	"testing"
)

func TestTheWorkersAskedForAreTheWorkersUsed(t *testing.T) {
	if got := getWorkerCount(3, 100); got != 3 {
		t.Errorf("%d workers share the work, want 3", got)
	}
}

func TestNoMoreWorkersThanJobsShareTheWork(t *testing.T) {
	if got := getWorkerCount(8, 2); got != 2 {
		t.Errorf("%d workers share 2 jobs, want 2", got)
	}
}

func TestAWorkerCountOfNoneTakesOneWorkerPerCPU(t *testing.T) {
	want := runtime.GOMAXPROCS(0)
	if got := getWorkerCount(0, want+10); got != want {
		t.Errorf("%d workers share the work, want %d", got, want)
	}
}

func TestAtLeastOneWorkerSharesTheWorkWhenThereAreNoJobs(t *testing.T) {
	if got := getWorkerCount(4, 0); got != 1 {
		t.Errorf("%d workers share no jobs, want 1", got)
	}
}

func TestTheFeaturesAreTheSameWhateverTheWorkerCount(t *testing.T) {
	signal := tone()
	want, frames := logMel(signal, 1)
	for _, workers := range []int{0, 2, 7} {
		got, made := logMel(signal, workers)
		if made != frames || !slices.Equal(got, want) {
			t.Errorf("%d workers made other features than one worker did", workers)
		}
	}
}

func TestTheResampledSignalIsTheSameWhateverTheWorkerCount(t *testing.T) {
	in := make([]float32, 4410)
	for i := range in {
		in[i] = float32(i%97) / 97
	}
	want, err := resample(t.Context(), in, 44100, sampleRate, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, workers := range []int{0, 3} {
		got, err := resample(t.Context(), in, 44100, sampleRate, workers)
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("%d workers resampled to another signal than one worker did (%v)", workers, err)
		}
	}
}
