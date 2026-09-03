package container_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// carrying is a container carrying one profile under a name.
func carrying(name string, profile proofreading.Profile) container.Config {
	cfg := proofreading.Defaults()
	cfg.Profiles = map[string]proofreading.Profile{name: profile}
	return container.Config{Proofreading: cfg}
}

// A container nobody configured proofreads with nothing, and the machine's own
// settings are not read here.
func TestAContainerNobodyGaveSettingsProofreadsWithNothing(t *testing.T) {
	proofreader, why := container.Config{}.Proofreader("", proofread.ScanInstruction)
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if proofreader != nil {
		t.Errorf("got a proofreader: %v", proofreader.Name())
	}
}

// A name no profile carries is said. A person who named one and is silently
// given nothing has no way to find out that nothing is proofreading.
func TestAProfileNameNoProfileCarriesIsRefused(t *testing.T) {
	held := carrying("openrouter", proofreading.ServiceDefaults())

	proofreader, why := held.Proofreader("agent", proofread.ScanInstruction)
	if why == nil {
		t.Fatal("no reason")
	}
	if !strings.Contains(why.Error(), "agent") {
		t.Errorf("the reason does not name the profile asked for: %v", why)
	}
	if proofreader != nil {
		t.Errorf("got a proofreader: %v", proofreader.Name())
	}
}

// A profile that asks for a service without saying which model is refused, as a
// name nobody carries is.
func TestAServiceWithoutAModelNameIsRefused(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "sk-test")

	_, why := carrying("openrouter", proofreading.ServiceDefaults()).
		Proofreader("openrouter", proofread.ScanInstruction)
	if why == nil {
		t.Fatal("no reason")
	}
	if !strings.Contains(why.Error(), "model") {
		t.Errorf("the reason does not say what is missing: %v", why)
	}
}

func TestAModelWithoutAKeyGivesTheReasonAndNoProofreader(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "")
	service := proofreading.ServiceDefaults()
	service.Name = "some-model"

	proofreader, why := carrying("openrouter", service).
		Proofreader("openrouter", proofread.ScanInstruction)
	if why == nil {
		t.Fatal("no reason")
	}
	if !strings.Contains(why.Error(), "key") {
		t.Errorf("the reason does not say what is missing: %v", why)
	}
	if proofreader != nil {
		t.Errorf("got a proofreader: %v", proofreader.Name())
	}
}

func TestTheProofreadingSettingsGivenAreTheOnesUsed(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "sk-test")
	service := proofreading.ServiceDefaults()
	service.Name = "some-model"
	// Nowhere: no test reaches a service.
	service.BaseURL = "http://127.0.0.1:1/v1"

	proofreader, why := carrying("openrouter", service).
		Proofreader("openrouter", proofread.ScanInstruction)
	if why != nil {
		t.Fatal(why)
	}
	if proofreader == nil {
		t.Fatal("no proofreader")
	}
	if got := proofreader.Name(); got != "some-model" {
		t.Errorf("got %q", got)
	}
}

// A profile reaching the command line is opened by the platform, which is told
// what to start and what the run is correcting.
func TestAnAgentProfileIsOpenedByThePlatform(t *testing.T) {
	profile := proofreading.AgentDefaults()
	profile.Model = "haiku"
	profile.Command = []string{"/somewhere/claude"}

	var asked container.AgentProofreader
	held := carrying("agent", profile)
	held.AgentProofreader = func(said container.AgentProofreader) (port.Proofreader, error) {
		asked = said
		return spelling{}, nil
	}

	proofreader, why := held.Proofreader("agent", proofread.SpeechInstruction)
	if why != nil {
		t.Fatal(why)
	}
	if proofreader == nil {
		t.Fatal("no proofreader")
	}
	if asked.Model != "haiku" || len(asked.Command) != 1 || asked.Command[0] != "/somewhere/claude" {
		t.Errorf("got %+v", asked)
	}
	if asked.Instruction != proofread.SpeechInstruction {
		t.Errorf("the instruction is %q", asked.Instruction)
	}
}

// An installation that starts no process names no profile that needs one.
func TestAnAgentProfileWithoutAPlatformIsRefused(t *testing.T) {
	profile := proofreading.AgentDefaults()
	profile.Model = "haiku"

	_, why := carrying("agent", profile).Proofreader("agent", proofread.ScanInstruction)
	if why == nil {
		t.Fatal("no reason")
	}
}

// A proofreader configured with a queue is one the pages can be left with.
func TestAProofreaderWithAQueueLeavesPagesWithIt(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "sk-a-key")
	service := proofreading.ServiceDefaults()
	service.Name = "test-model"

	queue, why := carrying("openrouter", service).
		ProofreadQueue("openrouter", proofread.ScanInstruction)
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if queue == nil || queue.Name() != "test-model" {
		t.Fatalf("got %v", queue)
	}
}

// A service with nowhere to leave the pages is asked a page at a time.
func TestAServiceWithoutAQueueLeavesNothing(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "sk-a-key")
	service := proofreading.ServiceDefaults()
	service.Name = "test-model"
	service.BatchURL = ""

	queue, why := carrying("openrouter", service).
		ProofreadQueue("openrouter", proofread.ScanInstruction)
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if queue != nil {
		t.Errorf("got a queue: %v", queue.Name())
	}
}

// A profile reaching the command line has no queue: it is asked and answers.
func TestAnAgentProfileLeavesNothing(t *testing.T) {
	profile := proofreading.AgentDefaults()
	profile.Model = "haiku"

	queue, why := carrying("agent", profile).ProofreadQueue("agent", proofread.ScanInstruction)
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if queue != nil {
		t.Errorf("got a queue: %v", queue.Name())
	}
}

// spelling is a proofreader that is never asked anything.
type spelling struct{}

func (spelling) Name() string { return "a test" }

func (spelling) Proofread(_ context.Context, _ []proofread.Batch) (map[int]string, error) {
	return nil, errors.New("nothing here asks")
}

// A profile reached through neither a service nor the command line is said to
// be, and the profile is named. A person who wrote a word this application does
// not know is owed the news.
func TestAProfileReachedThroughNeitherIsRefused(t *testing.T) {
	for _, use := range []string{"", "telepathy"} {
		profile := proofreading.Profile{Use: use, Name: "some-model", Model: "haiku"}
		held := carrying("mine", profile)
		held.AgentProofreader = func(container.AgentProofreader) (port.Proofreader, error) {
			return spelling{}, nil
		}

		proofreader, why := held.Proofreader("mine", proofread.ScanInstruction)
		if why == nil {
			t.Fatalf("a profile used through %q was opened", use)
		}
		for _, word := range []string{"mine", proofreading.UseService, proofreading.UseAgent} {
			if !strings.Contains(why.Error(), word) {
				t.Errorf("the reason does not name %q: %v", word, why)
			}
		}
		if proofreader != nil {
			t.Errorf("got a proofreader: %v", proofreader.Name())
		}
	}
}

// How much of a person's own model a run may take is the profile's to say, and
// it reaches the platform that starts the command line.
func TestHowManyRunsStandAtOnceReachesThePlatform(t *testing.T) {
	profile := proofreading.AgentDefaults()
	profile.Model = "haiku"
	profile.InFlight = 3

	var asked container.AgentProofreader
	held := carrying("agent", profile)
	held.AgentProofreader = func(said container.AgentProofreader) (port.Proofreader, error) {
		asked = said
		return spelling{}, nil
	}

	if _, why := held.Proofreader("agent", proofread.SpeechInstruction); why != nil {
		t.Fatal(why)
	}
	if asked.InFlight != 3 {
		t.Errorf("the platform was told %d runs may stand at once", asked.InFlight)
	}
}
