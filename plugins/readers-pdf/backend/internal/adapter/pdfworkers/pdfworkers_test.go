package pdfworkers

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
)

func TestTheProfileSetsTheIdleWorkers(t *testing.T) {
	for _, c := range []struct {
		profile performance.Profile
		cpus    int
		want    int
	}{
		{performance.Eco, 2, 1},
		{performance.Eco, 16, 1},
		{performance.Balanced, 4, 1},
		{performance.Balanced, 8, 2},
		{performance.Balanced, 32, 4},
		{performance.Maximum, 2, 1},
		{performance.Maximum, 8, 4},
		{performance.Maximum, 32, 8},
	} {
		if got := getIdleWorkers(c.profile, c.cpus); got != c.want {
			t.Errorf("%s on %d cores keeps %d idle workers, want %d", c.profile, c.cpus, got, c.want)
		}
	}
}
