package container

import (
	"testing"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

func makePlan(workers int) settingsdomain.ResolvedCompute {
	return settingsdomain.ResolvedCompute{Workers: workers}
}

func TestASearchExtractsAsManySourcesAtOnceAsThePlanHasWorkers(t *testing.T) {
	got := newSearchOver(searchOf(NewConfig(Config{
		Machine: Machine{ResolvedCompute: makePlan(5)},
	})), nil, nil, nil).Extractors
	if got != 5 {
		t.Errorf("a search extracts %d sources at once, want the 5 planned", got)
	}
}

func TestNewSearchOver_UsesDefaultExtractorCountWithoutPlan(t *testing.T) {
	want := search.New(nil, nil, nil, nil, nil, nil).Extractors
	if got := newSearchOver(searchOf((NewConfig(Config{}))), nil, nil, nil).Extractors; got != want {
		t.Errorf("a search extracts %d sources at once, want %d", got, want)
	}
}
