package pdfworkers

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

func TestTheProfileSetsTheWorkers(t *testing.T) {
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
		if got := getThreads(c.profile, c.cpus); got != c.want {
			t.Errorf("%s on %d cores runs %d workers, want %d", c.profile, c.cpus, got, c.want)
		}
	}
}

const ecoWorkers = 1

func TestConfigureSizesTheWorkersToTheProfile(t *testing.T) {
	workers := pdf.NewWorkers(pdf.Options{Report: ReportError, CompilationCache: nil, AfterFunc: nil, Workers: 0, IdleWorkers: 0})
	Configure(workers, map[string]string{performance.ConfigKey: string(performance.Eco)})
	if got := workers.GetWorkers(); got != ecoWorkers {
		t.Errorf("the eco profile runs %d workers, want %d", got, ecoWorkers)
	}
}
