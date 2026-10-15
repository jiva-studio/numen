package container

import (
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed/openai"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
)

func makePlan(workers, llm int) settings.ResolvedCompute {
	return settings.ResolvedCompute{Workers: workers, LlmConcurrency: llm}
}

func makeProofreading(inFlight int) ProofreadingConfig {
	service := ServiceDefaults()
	service.Name, service.InFlight = "some-model", inFlight
	agent := AgentDefaults()
	agent.Model, agent.InFlight = "haiku", inFlight
	cfg := DefaultProofreading()
	cfg.Profiles = map[string]Profile{"service": service, "agent": agent}
	return cfg
}

func TestAProofreadingProfileNamingANumberAtOnceKeepsItWhateverThePlanSays(t *testing.T) {
	cfg := Config{Proofreading: makeProofreading(3), ResolvedCompute: makePlan(8, 6)}
	for _, name := range []string{"service", "agent"} {
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
	for _, name := range []string{"service", "agent"} {
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
		ScanProofreading:       Proofread{Profile: "service"},
		TranscriptProofreading: Proofread{Profile: "agent"},
	}
	if got := scan.GetScanProofreading().InFlight; got != 6 {
		t.Errorf("a scan is proofread %d batches at once, want the 6 planned", got)
	}
	if got := scan.GetSpeechProofreading().InFlight; got != 6 {
		t.Errorf("speech is proofread %d batches at once, want the 6 planned", got)
	}
}

func TestAProofreadingProfileWithoutAPlanTakesTheDefaultOfItsKind(t *testing.T) {
	cfg := Config{Proofreading: makeProofreading(0)}
	service, err := cfg.getProfile("service")
	if err != nil {
		t.Fatal(err)
	}
	agent, err := cfg.getProfile("agent")
	if err != nil {
		t.Fatal(err)
	}
	if service.InFlight != DefaultInFlight || agent.InFlight != DefaultAgentInFlight {
		t.Errorf("a service asks about %d and the command line %d batches at once, want %d and %d",
			service.InFlight, agent.InFlight, DefaultInFlight, DefaultAgentInFlight)
	}
}

func newServiceModel(t *testing.T) (embed.ServiceModel, embed.Config) {
	t.Helper()
	t.Setenv(embed.KeyEnvVar, "sk-test")
	cfg := embed.Defaults()
	cfg.Model.Name = "bge-m3"
	cfg.Model.Dimensions = 4
	cfg.Indexing = newPlanServiceProvider(cfg.Indexing)
	service, _ := cfg.Indexing.Service()
	return service, cfg
}

func newPlanServiceProvider(where embed.Provider) embed.Provider {
	where.Use = embed.UseService
	service, _ := where.Service()
	service.Name, service.BaseURL = "bge-m3", "http://127.0.0.1:1/v1"
	return where.SetService(service)
}

func TestTheEmbeddingClientSendsAsManyRequestsAtOnceAsThePlanAllows(t *testing.T) {
	service, cfg := newServiceModel(t)
	client, err := Config{ResolvedCompute: makePlan(8, 3)}.openServiceClient(cfg.GetStoredModel(), service)
	if err != nil {
		t.Fatal(err)
	}
	if client.InFlight != 3 {
		t.Errorf("the client sends %d requests at once, want the 3 planned", client.InFlight)
	}
}

func TestTheEmbeddingClientOfAContainerWithoutAPlanKeepsItsOwnLimit(t *testing.T) {
	service, cfg := newServiceModel(t)
	is := cfg.GetStoredModel()
	own, err := openai.New(is, service)
	if err != nil {
		t.Fatal(err)
	}
	client, err := Config{}.openServiceClient(is, service)
	if err != nil {
		t.Fatal(err)
	}
	if client.InFlight != own.InFlight {
		t.Errorf("the client sends %d requests at once, want %d", client.InFlight, own.InFlight)
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
