package container

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/settings/hardware"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
)

func makePlan(workers, llm int) hardware.ResolvedCompute {
	return hardware.ResolvedCompute{Workers: workers}
}

func TestASearchExtractsAsManySourcesAtOnceAsThePlanHasWorkers(t *testing.T) {
	got := Config{ResolvedCompute: makePlan(5, 2)}.NewSearchOver(nil, nil, nil).GetExtractors()
	if got != 5 {
		t.Errorf("a search extracts %d sources at once, want the 5 planned", got)
	}
}

func TestASearchOfAContainerWithoutAPlanExtractsTheDefaultNumberOfSources(t *testing.T) {
	want := search.New(nil, nil, nil, nil, 0, nil).GetExtractors()
	if got := (Config{}).NewSearchOver(nil, nil, nil).GetExtractors(); got != want {
		t.Errorf("a search extracts %d sources at once, want %d", got, want)
	}
}
