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
	got := newSearchOver(searchOf(NewConfig(Config{ResolvedCompute: makePlan(5)})), nil, nil, nil).GetExtractors()
	if got != 5 {
		t.Errorf("a search extracts %d sources at once, want the 5 planned", got)
	}
}

func TestNewSearchOver_UsesDefaultExtractorCountWithoutPlan(t *testing.T) {
	want := search.New(nil, nil, nil, nil, nil, nil).GetExtractors()
	if got := newSearchOver(searchOf((NewConfig(Config{}))), nil, nil, nil).GetExtractors(); got != want {
		t.Errorf("a search extracts %d sources at once, want %d", got, want)
	}
}
