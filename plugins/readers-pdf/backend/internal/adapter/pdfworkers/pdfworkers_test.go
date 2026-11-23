package pdfworkers

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

func newTestWorkers() *pdf.Workers {
	return pdf.NewWorkers(pdf.Options{Report: ReportError, CompilationCache: nil, StartTimer: nil, Workers: 0, IdleWorkers: 0})
}

func TestConfigureSizesTheWorkersToTheProfile(t *testing.T) {
	for _, c := range []struct {
		profile performance.Profile
		threads string
		nice    string
		want    int
	}{
		{performance.Eco, "1", "19", 1},
		{performance.Low, "2", "15", 2},
		{performance.Balanced, "4", "10", 4},
		{performance.High, "6", "5", 6},
		{performance.Maximum, "8", "0", 8},
	} {
		workers := newTestWorkers()
		Configure(workers, map[string]string{
			performance.ProfileKey: string(c.profile),
			performance.ThreadsKey: c.threads,
			performance.NiceKey:    c.nice,
		})
		if got := workers.GetWorkers(); got != c.want {
			t.Errorf("%s with %s threads runs %d workers, want %d", c.profile, c.threads, got, c.want)
		}
	}
}

func TestConfigureWithoutLimitsRunsOneWorker(t *testing.T) {
	workers := newTestWorkers()
	Configure(workers, map[string]string{})
	if got := workers.GetWorkers(); got != 1 {
		t.Errorf("no limits run %d workers, want 1", got)
	}
}
