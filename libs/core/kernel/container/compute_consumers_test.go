package container

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings/hardware"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

func makePlan(workers int) hardware.ResolvedCompute {
	return hardware.ResolvedCompute{Workers: workers}
}

func TestASearchExtractsAsManySourcesAtOnceAsThePlanHasWorkers(t *testing.T) {
	got := newSearchOver(searchOf(NewConfig(Config{
		Machine: Machine{ResolvedCompute: makePlan(5)},
	})), search.Index{}, nil, nil).Extractors
	if got != 5 {
		t.Errorf("a search extracts %d sources at once, want the 5 planned", got)
	}
}

func TestNewSearchOver_UsesDefaultExtractorCountWithoutPlan(t *testing.T) {
	want := search.New(search.Index{}, nil, nil, nil, nil, nil).Extractors
	if got := newSearchOver(searchOf((NewConfig(Config{}))), search.Index{}, nil, nil).Extractors; got != want {
		t.Errorf("a search extracts %d sources at once, want %d", got, want)
	}
}
