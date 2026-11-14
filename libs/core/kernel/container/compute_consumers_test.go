package container

import (
	"testing"

	machinedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

func makePlan(workers int) machinedomain.ResolvedCompute {
	return machinedomain.ResolvedCompute{Workers: workers}
}

func TestASearchExtractsAsManySourcesAtOnceAsThePlanHasWorkers(t *testing.T) {
	cfg := NewConfig(Config{})
	cfg.machine = Machine{ResolvedCompute: makePlan(5)}
	got := newSearchOver(searchOf(cfg), nil, nil, nil, nil).Extractors
	if got != 5 {
		t.Errorf("a search extracts %d sources at once, want the 5 planned", got)
	}
}

func TestNewSearchOver_UsesDefaultExtractorCountWithoutPlan(t *testing.T) {
	want := search.New(nil, nil, nil, nil, nil, nil).Extractors
	if got := newSearchOver(searchOf((NewConfig(Config{}))), nil, nil, nil, nil).Extractors; got != want {
		t.Errorf("a search extracts %d sources at once, want %d", got, want)
	}
}
