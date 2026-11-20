package limits

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
)

const maximumBatchTexts = 8

func TestTheProfileSetsTheBatchAndTheHostSetsThreads(t *testing.T) {
	table := []struct {
		limits performance.Limits
		want   Run
	}{
		{performance.Limits{Profile: performance.Eco, Threads: 1, Nice: 19}, Run{Threads: 1, BatchTexts: 2}},
		{performance.Limits{Profile: performance.Low, Threads: 4, Nice: 15}, Run{Threads: 4, BatchTexts: 3}},
		{performance.Limits{Profile: performance.Balanced, Threads: 8, Nice: 10}, Run{Threads: 8, BatchTexts: 4}},
		{performance.Limits{Profile: performance.High, Threads: 12, Nice: 5}, Run{Threads: 12, BatchTexts: 6}},
		{performance.Limits{Profile: performance.Maximum, Threads: 16, Nice: 0}, Run{Threads: 16, BatchTexts: 8}},
	}
	for _, c := range table {
		if got := GetRun(c.limits, maximumBatchTexts); got != c.want {
			t.Errorf("%+v: %+v, want %+v", c.limits, got, c.want)
		}
	}
}

func TestHighNeverBatchesMoreThanMaximum(t *testing.T) {
	if got := GetRun(performance.Limits{Profile: performance.High, Threads: 2}, 5).BatchTexts; got != 5 {
		t.Errorf("High batches %d texts where Maximum batches 5", got)
	}
}

func TestNoLimitsFromTheHostRunOneThreadAtTheBalancedBatch(t *testing.T) {
	if got := GetRun(performance.Read(nil), maximumBatchTexts); got != (Run{Threads: 1, BatchTexts: 4}) {
		t.Errorf("the run without limits is %+v", got)
	}
}

func TestIndexShareReservesOnePassForQuestions(t *testing.T) {
	for _, tc := range []struct{ passes, want int }{{1, 1}, {2, 1}, {4, 3}} {
		if got := GetIndexShare(tc.passes); got != tc.want {
			t.Errorf("a model of %d passes gives the index %d, want %d", tc.passes, got, tc.want)
		}
	}
}
