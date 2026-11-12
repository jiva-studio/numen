package limits

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
)

const maximumBatchTexts = 8

func TestTheProfileSetsThreadsBatchAndPriorityOnEveryMachine(t *testing.T) {
	table := []struct {
		profile performance.Profile
		cpus    int
		want    Run
	}{
		{performance.Eco, 2, Run{Threads: 1, BatchTexts: 2, Nice: 19}},
		{performance.Eco, 16, Run{Threads: 1, BatchTexts: 2, Nice: 19}},
		{performance.Balanced, 2, Run{Threads: 1, BatchTexts: 4, Nice: 10}},
		{performance.Balanced, 4, Run{Threads: 1, BatchTexts: 4, Nice: 10}},
		{performance.Balanced, 8, Run{Threads: 2, BatchTexts: 4, Nice: 10}},
		{performance.Balanced, 16, Run{Threads: 4, BatchTexts: 4, Nice: 10}},
		{performance.Maximum, 2, Run{Threads: 1, BatchTexts: 8, Nice: 0}},
		{performance.Maximum, 4, Run{Threads: 2, BatchTexts: 8, Nice: 0}},
		{performance.Maximum, 8, Run{Threads: 4, BatchTexts: 8, Nice: 0}},
		{performance.Maximum, 16, Run{Threads: 8, BatchTexts: 8, Nice: 0}},
	}
	for _, c := range table {
		if got := GetRun(c.profile, c.cpus, maximumBatchTexts); got != c.want {
			t.Errorf("%s on %d cores: %+v, want %+v", c.profile, c.cpus, got, c.want)
		}
	}
}

func TestDefaultProfileUsesAtMostQuarterOfCores(t *testing.T) {
	for _, cpus := range []int{4, 8, 12, 16, 32} {
		if got := GetRun(performance.Read(nil), cpus, maximumBatchTexts).Threads; got*4 > cpus {
			t.Errorf("%d threads on %d cores", got, cpus)
		}
	}
}

func TestIndexShareReservesOnePassForQuestions(t *testing.T) {
	for _, tc := range []struct{ passes, want int }{{1, 1}, {2, 1}, {4, 3}} {
		if got := GetIndexShare(tc.passes); got != tc.want {
			t.Errorf("a model of %d passes gives the index %d, want %d", tc.passes, got, tc.want)
		}
	}
}
