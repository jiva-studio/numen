package container

import (
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
)

func makePlan(workers, llm int) settings.ResolvedCompute {
	return settings.ResolvedCompute{Workers: workers, LlmConcurrency: llm}
}

func makeProofreading(inFlight int) ProofreadingConfig {
	return ProofreadingConfig{Profiles: map[string]Profile{
		"careful": {InFlight: inFlight},
		"quick":   {InFlight: inFlight},
	}}
}

func TestAProofreadingProfileNamingANumberAtOnceKeepsItWhateverThePlanSays(t *testing.T) {
	cfg := Config{Proofreading: makeProofreading(3), ResolvedCompute: makePlan(8, 6)}
	for _, name := range []string{"careful", "quick"} {
		profile, err := cfg.getProfile(name)
		if err != nil {
			t.Fatal(err)
		}
		if profile.InFlight != 3 {
			t.Errorf("%s asks about %d batches at once, want the 3 it names", name, profile.InFlight)
		}
	}
}

func TestAProofreadingProfileNamingNoNumberAtOnceTakesTheLlmConcurrencyOfThePlan(t *testing.T) {
	cfg := Config{Proofreading: makeProofreading(0), ResolvedCompute: makePlan(8, 6)}
	for _, name := range []string{"careful", "quick"} {
		profile, err := cfg.getProfile(name)
		if err != nil {
			t.Fatal(err)
		}
		if profile.InFlight != 6 {
			t.Errorf("%s asks about %d batches at once, want the 6 planned", name, profile.InFlight)
		}
	}
	scan := Config{
		Proofreading:           cfg.Proofreading,
		ResolvedCompute:        cfg.ResolvedCompute,
		ScanProofreading:       Proofread{Profile: "careful"},
		TranscriptProofreading: Proofread{Profile: "quick"},
	}
	if got := scan.GetScanProofreading().InFlight; got != 6 {
		t.Errorf("a scan is proofread %d batches at once, want the 6 planned", got)
	}
	if got := scan.GetSpeechProofreading().InFlight; got != 6 {
		t.Errorf("speech is proofread %d batches at once, want the 6 planned", got)
	}
}

func TestAProofreadingProfileWithoutAPlanTakesTheDefault(t *testing.T) {
	cfg := Config{Proofreading: makeProofreading(0)}
	profile, err := cfg.getProfile("careful")
	if err != nil {
		t.Fatal(err)
	}
	if profile.InFlight != DefaultInFlight {
		t.Errorf("a profile asks about %d requests at once, want %d", profile.InFlight, DefaultInFlight)
	}
}

func TestASearchExtractsAsManySourcesAtOnceAsThePlanHasWorkers(t *testing.T) {
	got := Config{ResolvedCompute: makePlan(5, 2)}.NewSearchOver(nil, nil, nil).GetExtractors()
	if got != 5 {
		t.Errorf("a search extracts %d sources at once, want the 5 planned", got)
	}
}

func TestASearchOfAContainerWithoutAPlanExtractsTheDefaultNumberOfSources(t *testing.T) {
	want := search.New(nil, nil, nil, nil, nil, 0, nil).GetExtractors()
	if got := (Config{}).NewSearchOver(nil, nil, nil).GetExtractors(); got != want {
		t.Errorf("a search extracts %d sources at once, want %d", got, want)
	}
}

func TestTheCardPoolsRunAsManyPlacesAsThePlanHasWorkers(t *testing.T) {
	if got := (Config{ResolvedCompute: makePlan(6, 2)}).getCores(); got != 6 {
		t.Errorf("cards are parsed on %d places, want the 6 planned", got)
	}
}

func TestTheCardPoolsOfAContainerWithoutAPlanRunOnePlacePerCPU(t *testing.T) {
	if got, want := (Config{}).getCores(), runtime.GOMAXPROCS(0); got != want {
		t.Errorf("cards are parsed on %d places, want %d", got, want)
	}
}
